package arq

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/asimihsan/arqinator/arq/types"
	"io"
	"io/ioutil"
	"path"
)

type BlobKeysReader struct {
	blobKeys  []*arq_types.BlobKey
	apsi      *ArqPackSetIndex
	backupSet *ArqBackupSet
	bucket    *ArqBucket
	currentBlobKeyIndex int
	currentDataReader io.Reader
}

func GetReaderForBlobKeys(blobKeys []*arq_types.BlobKey, apsi *ArqPackSetIndex, backupSet *ArqBackupSet,
	bucket *ArqBucket) (*BlobKeysReader, error) {
	return &BlobKeysReader{
		blobKeys: blobKeys,
		apsi: apsi,
		backupSet: backupSet,
		bucket: bucket,
		currentBlobKeyIndex: 0,
	}, nil
}

func (r *BlobKeysReader) Read(p []byte) (int, error) {
	if r.currentDataReader != nil {
		return r.readChunk(p)
	}
	if r.currentBlobKeyIndex >= len(r.blobKeys) {
		return 0, io.EOF
	}
	blobKey := r.blobKeys[r.currentBlobKeyIndex]
	log.Debugf("node dataBlobKey: %s", blobKey)
	var contents []byte
	contents, err := r.apsi.GetBlobPackFile(r.backupSet, r.bucket, *blobKey.SHA1)
	if err != nil {
		log.Debugf("Couldn't find data in packfile, look at objects.")
		contents, err = GetDataBlobKeyContentsFromObjects(*blobKey.SHA1, r.bucket)
		if err != nil {
			err2 := errors.New(fmt.Sprintf("Couldn't find SHA %s in packfile or objects!",
				hex.EncodeToString((*blobKey.SHA1)[:])))
			log.Debugf("%s", err2)
			return 0, err2
		}
	}
	log.Debugf("len(contents): %d", len(contents))
	r.currentDataReader = bytes.NewReader(contents)
	return r.readChunk(p)
}

func (r *BlobKeysReader) readChunk(p []byte) (int, error) {
	n, err := r.currentDataReader.Read(p)
	if err == io.EOF {
		r.currentDataReader = nil
		r.currentBlobKeyIndex++
		err = nil
	}
	return n, err
}

func GetDataBlobKeyContentsFromObjects(SHA1 [20]byte, bucket *ArqBucket) ([]byte, error) {
	SHA1String := hex.EncodeToString(SHA1[:])
	log.Debugf("GetDataBlobKeyContents SHA1 %s, bucket %s", SHA1String, bucket)
	backupSet := bucket.ArqBackupSet
	key := path.Join(backupSet.UUID, "objects", SHA1String)
	log.Debugf("key: %s", key)
	dataFilepath, err := backupSet.Connection.CachedGet(key)
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
