package arq

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/mattn/go-plist"

	"github.com/asimihsan/arqinator/connector"
	"github.com/asimihsan/arqinator/crypto"
	"strings"
)

type ArqBackupSet struct {
	Connection      connector.Connection
	UUID            string
	ComputerInfo    *ArqComputerInfo
	Buckets         []*ArqBucket
	BlobDecrypter   *crypto.CryptoState
	BucketDecrypter *crypto.CryptoState
}

func GetArqBackupSets(connection connector.Connection, password []byte) ([]*ArqBackupSet, error) {
	prefix := ""
	objects, err := connection.ListObjectsAsFolders(prefix)
	if err != nil {
		log.Debugln("Failed to get buckets for GetArqBackupSets: ", err)
		return nil, err
	}
	arqBackupSets := make([]*ArqBackupSet, 0)
	for _, object := range objects {
		arqBackupSet, err := NewArqBackupSet(connection, password, object.GetPath())
		if err != nil {
			log.Debugf("Error during GetArqBackupSets for object %s: %s", object, err)
			continue
		}
		arqBackupSets = append(arqBackupSets, arqBackupSet)
	}
	return arqBackupSets, nil
}

func NewArqBackupSet(connection connector.Connection, password []byte, uuid string) (*ArqBackupSet, error) {
	var err error
	abs := ArqBackupSet{
		Connection:   connection,
		UUID:         uuid,
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
	return fmt.Sprintf("{ArqBackupSet: Connection=%s, UUID=%s, ComputerInfo=%s, Buckets=%s}",
		abs.Connection, abs.UUID, abs.ComputerInfo, abs.Buckets)
}

type ArqComputerInfo struct {
	UserName     string
	ComputerName string
}

func (aci ArqComputerInfo) String() string {
	return fmt.Sprintf("{ArqComputerInfo: UserName=%s, ComputerName=%s}", aci.UserName, aci.ComputerName)
}

func (abs *ArqBackupSet) getSalt() ([]byte, error) {
	key := abs.UUID + "/salt"
	filepath, err := abs.Connection.CachedGet(key)
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
	key := abs.UUID + "/computerinfo"
	filepath, err := abs.Connection.CachedGet(key)
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

func (abs *ArqBackupSet) CacheBlobPackSets() error {
	log.Debugln("CacheBlobPackSets entry for ArqBackupSet: ", abs)
	defer log.Debugln("CacheBlobPackSets exit for ArqBackupSet: ", abs)
	for i := range abs.Buckets {
		abs.cacheBlobPackSet(abs.Buckets[i])
	}
	return nil
}

func (abs *ArqBackupSet) cacheBlobPackSet(ab *ArqBucket) error {
	prefix := GetPathToBucketPackSetBlobs(abs, ab)
	return abs.cachePackSet(ab, prefix)
}

func (abs *ArqBackupSet) cacheTreePackSet(ab *ArqBucket) error {
	prefix := GetPathToBucketPackSetTrees(abs, ab)
	return abs.cachePackSet(ab, prefix)
}

func (abs *ArqBackupSet) cachePackSet(ab *ArqBucket, prefix string) error {
	s3Objs, err := abs.Connection.ListObjectsAsAll(prefix)
	if err != nil {
		log.Debugln("Failed to cacheTreePackSet for bucket: ", ab)
		log.Debugln(err)
		return err
	}
	inputs := make(chan connector.Object, len(s3Objs))
	for i := range s3Objs {
		inputs <- s3Objs[i]
	}
	close(inputs)
	c := make(chan int, runtime.GOMAXPROCS(0)*2)
	for i := 0; i < cap(c); i++ {
		go func() {
			for inputObject := range inputs {
				if strings.HasSuffix(inputObject.GetPath(), ".index") {
					_, err := abs.Connection.CachedGet(inputObject.GetPath())
					if err != nil {
						log.Debugln("cacheTreePackSet failed to get object: ", inputObject)
						log.Debugln(err)
					}
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
	prefix := abs.UUID + "/buckets"
	objects, err := abs.Connection.ListObjectsAsAll(prefix)
	if err != nil {
		log.Debugln("Failed to get buckets for ArqBackupSet: ", err)
		return nil, err
	}
	buckets := make([]*ArqBucket, 0)
	for _, object := range objects {
		bucket, err := NewArqBucket(object, abs)
		if err != nil {
			log.Debugln("Failed to get ArqBucket for object: ", object)
			log.Debugln(err)
			return nil, err
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}
