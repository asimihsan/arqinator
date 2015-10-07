package arq

import (
	"fmt"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"
	"os"
	"runtime"

	"github.com/mattn/go-plist"

	"github.com/asimihsan/arqinator/connector"
	"github.com/asimihsan/arqinator/crypto"
)

type ArqBackupSet struct {
	S3BucketName    string
	Connection      *connector.S3Connection
	Uuid            string
	ComputerInfo    *ArqComputerInfo
	Buckets         []*ArqBucket
	BlobDecrypter   *crypto.CryptoState
	BucketDecrypter *crypto.CryptoState
}

func GetArqBackupSets(s3BucketName string, connection *connector.S3Connection, password []byte) ([]*ArqBackupSet, error) {
	prefix := ""
	s3Objs, err := connection.ListObjectsAsFolders(s3BucketName, prefix)
	if err != nil {
		log.Debugln("Failed to get buckets for GetArqBackupSets: ", err)
		return nil, err
	}
	arqBackupSets := make([]*ArqBackupSet, 0)
	for _, s3Obj := range s3Objs {
		arqBackupSet, err := NewArqBackupSet(s3BucketName, connection, password, s3Obj.S3FullPath)
		if err != nil {
			log.Debugf("Error during GetArqBackupSets for s3Obj %s: %s", s3Obj, err)
			continue
		}
		arqBackupSets = append(arqBackupSets, arqBackupSet)
	}
	return arqBackupSets, nil
}

func NewArqBackupSet(s3BucketName string, connection *connector.S3Connection, password []byte, uuid string) (*ArqBackupSet, error) {
	var err error
	abs := ArqBackupSet{
		S3BucketName: s3BucketName,
		Connection:   connection,
		Uuid:         uuid,
	}

	// Regular objects (commits, trees, blobs) use a random "salt" stored in backup
	var salt []byte
	if salt, err = abs.getSalt(); err != nil {
		log.Debugln("Failed during NewArqBackupSet getSalt: ", err)
		return nil, err
	}
	if abs.BlobDecrypter, err = crypto.NewCryptoState(password, salt); err != nil {
		log.Debugln("Failed during NewArqBackupSet NewCryptoState for BlobDecrypter: ", err)
		return nil, err
	}

	// Arq Buckets (the folders) use a fixed salt. See arq_restore/Bucket.m.
	if abs.BucketDecrypter, err = crypto.NewCryptoState(password, []byte("BucketPL")); err != nil {
		log.Debugln("Failed during NewArqBackupSet NewCryptoState for BucketDecrypter: ", err)
		return nil, err
	}

	if abs.ComputerInfo, err = abs.getComputerInfo(); err != nil {
		log.Debugln("Failed during NewArqBackupSet getComputerInfo: ", err)
		return nil, err
	}

	if abs.Buckets, err = abs.getBuckets(); err != nil {
		log.Debugln("Failed during NewArqBackupSet getBuckets: ", err)
		return nil, err
	}

	return &abs, nil
}

func (abs ArqBackupSet) String() string {
	return fmt.Sprintf("{ArqBackupSet: S3BucketName=%s, Uuid=%s, ComputerInfo=%s, Buckets=%s}",
		abs.S3BucketName, abs.Uuid, abs.ComputerInfo, abs.Buckets)
}

type ArqComputerInfo struct {
	UserName     string
	ComputerName string
}

func (aci ArqComputerInfo) String() string {
	return fmt.Sprintf("{ArqComputerInfo: UserName=%s, ComputerName=%s}", aci.UserName, aci.ComputerName)
}

func (abs *ArqBackupSet) getSalt() ([]byte, error) {
	key := abs.Uuid + "/salt"
	filepath, err := abs.Connection.CachedGet(abs.S3BucketName, key)
	if err != nil {
		log.Debugln("Failed to get salt", err)
		return nil, err
	}
	salt, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Debugln("Failed to read salt from file: ", err)
		return nil, err
	}
	return salt, err
}

func (abs *ArqBackupSet) getComputerInfo() (*ArqComputerInfo, error) {
	key := abs.Uuid + "/computerinfo"
	filepath, err := abs.Connection.CachedGet(abs.S3BucketName, key)
	if err != nil {
		log.Debugln("Failed to get computerinfo", err)
		return nil, err
	}
	r, err := os.Open(filepath)
	if err != nil {
		log.Debugln("Failed to open computerinfo on disk")
		return nil, err
	}
	defer r.Close()
	v, err := plist.Read(r)
	if err != nil {
		log.Debugln("Could not decode computerInfo", err)
		return nil, err
	}
	tree := v.(plist.Dict)
	return &ArqComputerInfo{
		UserName:     tree["userName"].(string),
		ComputerName: tree["computerName"].(string),
	}, nil
}

func (abs *ArqBackupSet) CacheTreePackSets() error {
	log.Debugln("CacheTreePackSets entry for ArqBackupSet: ", abs)
	defer log.Debugln("CacheTreePackSets exit for ArqBackupSet: ", abs)
	for i := range abs.Buckets {
		abs.cacheTreePackSet(abs.Buckets[i])
	}
	return nil
}

func (abs *ArqBackupSet) cacheTreePackSet(ab *ArqBucket) error {
	prefix := GetPathToBucketPackSetTrees(abs, ab)
	s3Objs, err := abs.Connection.ListObjectsAsAll(abs.S3BucketName, prefix)
	if err != nil {
		log.Debugln("Failed to cacheTreePackSet for bucket: ", ab)
		log.Debugln(err)
		return err
	}
	inputs := make(chan *connector.S3Object, len(s3Objs))
	for i := range s3Objs {
		inputs <- &s3Objs[i]
	}
	close(inputs)
	c := make(chan int, runtime.GOMAXPROCS(0))
	for i := 0; i < cap(c); i++ {
		go func() {
			for inputS3Obj := range inputs {
				_, err := abs.Connection.CachedGet(abs.S3BucketName, inputS3Obj.S3FullPath)
				if err != nil {
					log.Debugln("cacheTreePackSet failed to get S3 object: ", inputS3Obj)
					log.Debugln(err)
				}
			}
			c <- 1
		}()
	}
	for i := 0; i < cap(c); i++ {
		<-c
	}
	return nil
}

func (abs *ArqBackupSet) getBuckets() ([]*ArqBucket, error) {
	prefix := abs.Uuid + "/buckets"
	s3Objs, err := abs.Connection.ListObjectsAsAll(abs.S3BucketName, prefix)
	if err != nil {
		log.Debugln("Failed to get buckets for ArqBackupSet: ", err)
		return nil, err
	}
	buckets := make([]*ArqBucket, 0)
	for i := range s3Objs {
		bucket, err := NewArqBucket(&s3Objs[i], abs)
		if err != nil {
			log.Debugln("Failed to get ArqBucket for s3Obj: ", s3Objs[i])
			log.Debugln(err)
			return nil, err
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}
