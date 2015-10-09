package arq

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/asimihsan/arqinator/arq/types"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func getWriterForFile(destinationPath string, mode os.FileMode, size int64) (*os.File, *bufio.Writer, error) {
	f, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		log.Errorf("Failed to open destinationPath %s: %s", destinationPath, err)
		return nil, nil, err
	}
	err = f.Truncate(int64(size))
	if err != nil {
		log.Errorf("Failed to pre-allocate size of file %s: %s", destinationPath, err)
		return nil, nil, err
	}
	w := bufio.NewWriter(f)
	return f, w, nil
}

func DownloadNode(node *arq_types.Node, cacheDirectory string, backupSet *ArqBackupSet,
	bucket *ArqBucket, sourcePath string, destinationPath string) error {
	log.Debugf("DownloadNode entry. sourcePath: %s, destinationPath: %s, node: %s", sourcePath, destinationPath, node)
	apsi, _ := NewPackSetIndex(cacheDirectory, backupSet, bucket)
	f, w, err := getWriterForFile(destinationPath, node.Mode, int64(node.UncompressedDataSize))
	if err != nil {
		log.Errorf("Failed during NewWriterForFile for node %s: %s", node, err)
		return err
	}
	defer f.Close()
	defer w.Flush()
	r, err := getReaderForBlobKeys(node.DataBlobKeys, apsi, backupSet, bucket)
	if err != nil {
		log.Errorf("Failed during GetReaderForBlobKeys for node %s: %s", node, err)
		return err
	}
	io.Copy(w, r)
	log.Debugf("DownloadNode exit. destinationPath: %s, node: %s", destinationPath, node)
	return nil
}

func DownloadTree(tree *arq_types.Tree, cacheDirectory string, backupSet *ArqBackupSet,
	bucket *ArqBucket, sourcePath string, destinationPath string) error {
	log.Debugf("DownloadTree entry. sourcePath: %s, destinationPath: %s, tree: %s", sourcePath, destinationPath, tree)
	if err := os.MkdirAll(destinationPath, tree.Mode); err != nil {
		log.Errorf("DownloadTree failed during MkdirAll %s: %s", destinationPath, err)
	}
	for _, node := range tree.Nodes {
		subSourcePath := filepath.Join(sourcePath, string(node.Name.Data))
		subDestinationPath := filepath.Join(destinationPath, string(node.Name.Data))
		subTree, subNode, err := FindNode(cacheDirectory, backupSet, bucket, subSourcePath)
		if err != nil {
			log.Errorf("DownloadTree failed FindNode: %s", err)
			return err
		}
		if subNode.IsTree.IsTrue() {
			err = DownloadTree(subTree, cacheDirectory, backupSet, bucket, subSourcePath, subDestinationPath)
		} else {
			err = DownloadNode(subNode, cacheDirectory, backupSet, bucket, subSourcePath, subDestinationPath)
		}
		if err != nil {
			log.Errorf("DownloadTree failed during subNode %s: %s. Will continue!", subNode, err)
		}
	}
	log.Debugf("DownloadTree exit. destinationPath: %s, tree: %s", destinationPath, tree)
	return nil
}

type BlobKeysReader struct {
	blobKeys            []*arq_types.BlobKey
	apsi                *ArqPackSetIndex
	backupSet           *ArqBackupSet
	bucket              *ArqBucket
	currentBlobKeyIndex int
	currentDataReader   io.Reader
}

func getReaderForBlobKeys(blobKeys []*arq_types.BlobKey, apsi *ArqPackSetIndex, backupSet *ArqBackupSet,
	bucket *ArqBucket) (*BlobKeysReader, error) {
	return &BlobKeysReader{
		blobKeys:            blobKeys,
		apsi:                apsi,
		backupSet:           backupSet,
		bucket:              bucket,
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
