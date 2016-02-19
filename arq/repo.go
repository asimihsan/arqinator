/*
arqinator: arq/types/repo.go
Implements an Arq Repo, a way of retrieving files and folders.

Copyright 2015 Asim Ihsan

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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

var (
	ErrorCouldNotRecoverTree = errors.New("Couldn't find a tree in the Arq backup")
)
/*
If we're running on Windows then convert a path to a Windows version of it.
e.g. on Windows /C/Users/username becomes C:\Users\username
however on e.g. Mac /C/Users/username returns /C/Users/username
 */
func maybeConvertToWindowsPath(path string) string {
	newPath := filepath.FromSlash(path)
	if newPath == path {
		return path
	}
	newPath = fmt.Sprintf("%s:%s", string(newPath[1]), string(newPath[2:]))
	log.Debugf("maybeConvertToWindowsPath converted %s to %s", path, newPath)
	return newPath
}

/**
	This function is not responsible for closing the file handle you pass in.
 */
func getWriterForFile(destinationPath string, mode os.FileMode, size int64) (*os.File, *bufio.Writer, error) {
	destinationPath = maybeConvertToWindowsPath(destinationPath)
	f, err := os.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		log.Errorf("getWriterForFile failed to open destinationPath %s: %s", destinationPath, err)
		return nil, nil, err
	}
	log.Debugf("getWriterForFile is writing to: %s", destinationPath)
	err = f.Truncate(int64(size))
	if err != nil {
		log.Errorf("getWriterForFile failed to pre-allocate size of file %s: %s", destinationPath, err)
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
		log.Errorf("Failed during DownloadNode getWriterForFile for node %s: %s", node, err)
		return err
	}
	defer f.Close()
	defer w.Flush()
	r, err := getReaderForBlobKeys(node.DataBlobKeys, apsi, backupSet, bucket)
	if err != nil {
		log.Errorf("Failed during DownloadNode GetReaderForBlobKeys for node %s: %s", node, err)
		return err
	}
	io.Copy(w, r)
	log.Debugf("DownloadNode exit. destinationPath: %s, node: %s", destinationPath, node)
	return nil
}

func DownloadTree(tree *arq_types.Tree, cacheDirectory string, backupSet *ArqBackupSet,
	bucket *ArqBucket, sourcePath string, destinationPath string) error {
	log.Debugf("DownloadTree entry. sourcePath: %s, destinationPath: %s, tree: %s", sourcePath, destinationPath, tree)

	// if tree is null we failed to find this part of the directory structure in Arq. Either we didn't back it up
	// or something really wrong happened whilst trying to recover it. Warn loudly but don't stop the backup,
	// because there may still be other files we can recover!
	if (tree == nil) {
		log.Warnf("DownloadTree: couldn't find sourcePath %s in backup, hence cannot recover it. Will continue recovering other files.", sourcePath)
		return ErrorCouldNotRecoverTree
	}

	directoryToCreate := maybeConvertToWindowsPath(destinationPath)
	if err := os.Mkdir(directoryToCreate, tree.Mode); err != nil {
		log.Errorf("DownloadTree failed during MkdirAll %s: %s", directoryToCreate, err)
	}
	if tree.Mode == os.FileMode(int(0)) {
		log.Debugf("tree %s isn't readable or writeable by anyone, fix up", tree)
		if err := os.Chmod(directoryToCreate, os.FileMode(int(0775))); err != nil {
			log.Errorf("Failed to set permissions of tree %s: %s", tree, err)
		}
	}
	for _, node := range tree.Nodes {
		subSourcePath := path.Join(sourcePath, string(node.Name.Data))
		subDestinationPath := path.Join(destinationPath, string(node.Name.Data))
		subTree, subNode, err := FindNode(cacheDirectory, backupSet, bucket, subSourcePath)
		if err != nil {
			log.Errorf("DownloadTree failed FindNode: %s", err)
			return err
		}
		if subNode == nil || subNode.IsTree.IsTrue() {
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
