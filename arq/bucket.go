package arq

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"
	"path"

	"github.com/mattn/go-plist"

	"github.com/asimihsan/arqinator/connector"
)

type ArqBucket struct {
	S3Obj        *connector.S3Object
	UUID         string
	LocalPath    string
	ArqBackupSet *ArqBackupSet
	HeadSHA1     [20]byte
}

func (ab ArqBucket) String() string {
	return fmt.Sprintf("{ArqBucket: UUID=%s, LocalPath=%s, S3Obj=%s, HeadSHA1=%s}",
		ab.UUID, ab.LocalPath, ab.S3Obj, hex.EncodeToString(ab.HeadSHA1[:]))
}

func (ab *ArqBucket) parsePlist() error {
	filepath, err := ab.ArqBackupSet.Connection.CachedGet(
		ab.ArqBackupSet.S3BucketName, ab.S3Obj.S3FullPath)
	if err != nil {
		log.Debugln("Failed during NewArqBucket for s3Obj: ", ab.S3Obj)
		log.Debugln(err)
		return err
	}
	bucket_encrypted, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Debugf("Failed during ArqBucket (%s) NewArqBucket: %s", ab, err)
		return err
	}
	bucket_decrypted, err := ab.ArqBackupSet.BucketDecrypter.Decrypt(bucket_encrypted)
	if err != nil {
		err2 := errors.New(fmt.Sprintf("Failed to decrypt bucket: %s", err))
		log.Debugf("%s", err2)
		return err2
	}
	bucket_data, err := plist.Read(bytes.NewReader(bucket_decrypted))
	if err != nil {
		log.Debugln("Couldn't parse bucket into plist: ", ab)
		log.Debugln(err)
		return err
	}
	tree := bucket_data.(plist.Dict)
	ab.LocalPath = tree["LocalPath"].(string)
	return nil
}

func assignSHA1(source []byte, destination *[20]byte) error {
	if len(source) != len(destination) {
		log.Debugf("Source %x not expected length %d", source, len(destination))
		return errors.New("Source SHA1 byte array not expected length")
	}
	for i, b := range source {
		destination[i] = b
	}
	return nil
}

func (ab *ArqBucket) updateHeadSHA1() error {
	key := path.Join(ab.ArqBackupSet.Uuid, "bucketdata", ab.UUID,
		"refs", "heads", "master")
	filepath, err := ab.ArqBackupSet.Connection.Get(
		ab.ArqBackupSet.S3BucketName, key)
	if err != nil {
		log.Debugf("Failed during ArqBucket (%s) updateHeadSHA1 get: %s",
			ab, err)
		return err
	}
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Debugf("Failed during ArqBucket (%s) updateHeadSHA1 readFile: %s",
			ab, err)
		return err
	}
	data = bytes.TrimSuffix(data, []byte("Y"))
	dataDecoded, err := hex.DecodeString(string(data))
	if err != nil {
		log.Debugf("Could not decode HeadSHA1 %s as hex: %s", data, err)
		return err
	}
	err = assignSHA1(dataDecoded, &ab.HeadSHA1)
	if err != nil {
		log.Debugf("Failed to record HEAD SHA1 for bucket %s: %s", ab, err)
		return err
	}
	return nil
}

func NewArqBucket(s3Obj *connector.S3Object, abs *ArqBackupSet) (*ArqBucket, error) {
	bucket := ArqBucket{S3Obj: s3Obj, ArqBackupSet: abs}
	bucket.UUID = path.Base(s3Obj.S3FullPath)
	err := bucket.parsePlist()
	if err != nil {
		err2 := errors.New(fmt.Sprintf("Failed during NewArqBucket: %s", err))
		log.Debugf("%s", err2)
		return nil, err2
	}
	bucket.updateHeadSHA1()
	return &bucket, nil
}
