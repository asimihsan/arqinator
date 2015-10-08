package arq
import (
	"encoding/hex"
	log "github.com/Sirupsen/logrus"
	"path"
	"errors"
	"fmt"
	"io/ioutil"
	"bytes"
	"compress/gzip"
	"io"
)

func GetDataBlobKeyContentsFromObjects(SHA1 [20]byte, bucket *ArqBucket) ([]byte, error) {
	SHA1String := hex.EncodeToString(SHA1[:])
	log.Debugf("GetDataBlobKeyContents SHA1 %s, bucket %s", SHA1String, bucket)
	backupSet := bucket.ArqBackupSet
	key := path.Join(backupSet.Uuid, "objects", SHA1String)
	log.Debugf("key: %s", key)
	dataFilepath, err := backupSet.Connection.CachedGet(backupSet.S3BucketName, key)
	if err != nil {
		err2 := errors.New(fmt.Sprintf("downloadDataFromDataBlobKey: failed to download SHA1 %s: %s", SHA1String, err))
		log.Errorf("%s", err2)
		return nil, err2
	}

	encrypted, err := ioutil.ReadFile(dataFilepath)
	if err != nil {
		log.Debugf("downloadDataFromDataBlobKey: Failed to read dataFilepath: %s", dataFilepath)
		return nil, err
	}

	decrypted, err := backupSet.BlobDecrypter.Decrypt(encrypted)
	if err != nil {
		log.Debugf("downloadDataFromDataBlobKey failed to decrypt: %s", err)
		return nil, err
	}
	// Try to decompress, if fails then assume it was uncompressed to begin with
	var b bytes.Buffer
	r, err := gzip.NewReader(bytes.NewBuffer(decrypted))
	if err != nil {
		log.Debugf("downloadDataFromDataBlobKey decompression failed during NewReader, assume not compresed: ", err)
		return nil, err
	}
	if _, err = io.Copy(&b, r); err != nil {
		log.Debugf("downloadDataFromDataBlobKey decompression failed during io.Copy, assume not compresed: ", err)
		return nil, err
	}
	if err := r.Close(); err != nil {
		log.Debugf("downloadDataFromDataBlobKey decompression failed during reader Close, assume not compresed: ", err)
		return nil, err
	}
	return b.Bytes(), nil
}
