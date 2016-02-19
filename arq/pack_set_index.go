/*
arqinator: arq/pack_set_index.go
Implements an Arq PackSetIndex, a way of getting packed blobs, trees, and commits.

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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"unsafe"

	"compress/gzip"
	"encoding/hex"
	"github.com/asimihsan/arqinator/arq/types"
	"io/ioutil"
	"strings"
)

type ArqPackSetIndex struct {
	CacheDirectory string
	ArqBackupSet   *ArqBackupSet
	ArqBucket      *ArqBucket
}

func GetPathToBucketPackSetTrees(abs *ArqBackupSet, ab *ArqBucket) string {
	return path.Join(abs.UUID, "packsets", fmt.Sprintf("%s-trees", ab.UUID))
}

func GetPathToBucketPackSetBlobs(abs *ArqBackupSet, ab *ArqBucket) string {
	return path.Join(abs.UUID, "packsets", fmt.Sprintf("%s-blobs", ab.UUID))
}

func NewPackSetIndex(cacheDirectory string, abs *ArqBackupSet, ab *ArqBucket) (*ArqPackSetIndex, error) {
	apsi := ArqPackSetIndex{
		CacheDirectory: cacheDirectory,
		ArqBackupSet:   abs,
		ArqBucket:      ab,
	}
	return &apsi, nil
}

func (apsi ArqPackSetIndex) String() string {
	return fmt.Sprintf("{ArqPackSetIndex: CacheDirectory=%s, ArqBucket=%s}",
		apsi.CacheDirectory, apsi.ArqBucket)
}

func FindNode(cacheDirectory string, backupSet *ArqBackupSet, bucket *ArqBucket, targetPath string) (*arq_types.Tree, *arq_types.Node, error) {
	apsi, _ := NewPackSetIndex(cacheDirectory, backupSet, bucket)
	commit, err := apsi.GetPackFileAsCommit(backupSet, bucket, bucket.HeadSHA1)
	if err != nil {
		err := errors.New(fmt.Sprintf("failed to get commit: %s", err))
		log.Debugf("%s", err)
		return nil, nil, err
	}
	log.Debugf("commit: %s", commit)
	if !strings.HasPrefix(targetPath, commit.Path) {
		err := errors.New(fmt.Sprintf("Target path %s is not located within commit path %s", targetPath, commit.Path))
		log.Errorf("%s", err)
		return nil, nil, err
	}

	var currentNode *arq_types.Node
	var tree *arq_types.Tree
	currentHash := commit.TreeBlobKey.SHA1
	currentPath := commit.Path
	for {
		log.Debugf("currentPath: %s, currentHash: %s", currentPath, hex.EncodeToString((*currentHash)[:]))

		nextPathElement := strings.TrimPrefix(targetPath, currentPath)
		nextPathElement = path.Clean(nextPathElement)
		if nextPathElement == "/" {
			nextPathElement = "."
		}
		nextPathElement = strings.TrimPrefix(nextPathElement, "/")
		nextPathElement = strings.Split(nextPathElement, "/")[0]
		log.Debugf("FindNode nextPathElement: %s", nextPathElement)

		tree, err = apsi.GetPackFileAsTree(backupSet, bucket, *currentHash)
		if err != nil {
			log.Debugf("failed to get tree: %s", err)
		}

		if nextPathElement == "." {
			log.Debugf("found node: %s", currentNode)
			return tree, currentNode, nil
		}

		found := false
		for _, node := range tree.Nodes {
			if node.Name.Equal(nextPathElement) {
				found = true
				currentNode = node
				currentHash = node.DataBlobKeys[0].SHA1
			}
		}
		if found == false {
			err := errors.New(fmt.Sprintf("Failed to find targetPath: %s", targetPath))
			log.Debugf("%s", err)
			return nil, nil, err
		}
		currentPath = path.Join(currentPath, nextPathElement)
	}
}

func (apsi *ArqPackSetIndex) GetPackFileAsCommit(backupSet *ArqBackupSet, bucket *ArqBucket, SHA1 [20]byte) (*arq_types.Commit, error) {
	pf, err := apsi.GetTreePackFile(backupSet, bucket, bucket.HeadSHA1)
	if err != nil {
		log.Debugf("GetPackFileAsCommit failed during apsi.GetPackFile: ", err)
		return nil, err
	}
	commit, err := arq_types.ReadCommit(bytes.NewBuffer(pf))
	if err != nil {
		log.Debugf("failed to parse commit: %s", err)
	}
	return commit, nil
}

func (apsi *ArqPackSetIndex) GetPackFileAsTree(backupSet *ArqBackupSet, bucket *ArqBucket, SHA1 [20]byte) (*arq_types.Tree, error) {
	log.Debugf("get tree_packfile...")
	tree_packfile, err := apsi.GetTreePackFile(backupSet, bucket, SHA1)
	if err != nil {
		log.Debugf("failed to get tree blob: %s", err)
		return nil, err
	}
	log.Debugf("finished getting tree_packfile.")

	log.Debugf("get tree...")
	tree, err := arq_types.ReadTree(bytes.NewBuffer(tree_packfile))
	if err != nil {
		log.Debugf("failed to get tree: %s", err)
		return nil, err
	}
	log.Debugf("finished getting tree.")
	log.Debugf("tree: %s", tree)
	return tree, nil
}

type PackIndex struct {
	_      uint32
	_      uint32
	Fanout [256]uint32
}

type PackIndexObject struct {
	Offset uint64
	Length uint64
	SHA1   [20]byte
	_      uint32
}

func (pio PackIndexObject) String() string {
	return fmt.Sprintf("{PackIndexObject: Offset=%d, Length=%d, SHA1=%x}",
		pio.Offset, pio.Length, pio.SHA1)
}

func readIntoPackIndexObject(r io.Reader) (*PackIndexObject, error) {
	pio := PackIndexObject{}
	err := binary.Read(r, binary.BigEndian, &pio)
	if err != nil {
		log.Debugf("Failed during readIntoPackIndexObject: %s", err)
		return nil, err
	}
	return &pio, nil
}

func testEq(a [20]byte, b [20]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

/*
	inputs := make(chan string, len(indexes))
	for i := range indexes {
		inputs <- indexes[i]
	}
	close(inputs)
	result := make(chan
	c := make(chan int, runtime.GOMAXPROCS(0))
	for i := 0; i < cap(c); i++ {
		go func() {
			var header PackIndex
			var pio PackIndexObject
			for index := range inputs {
				file, err := os.OpenFile(index, os.O_RDONLY, 0644)
				if err != nil {
					log.Panicln("Couldn't open index file: ", err)
				}
				defer file.Close()
				mmap, err := mmap.Map(file, mmap.RDONLY, 0)
				if err != nil {
					log.Panicln("Failed to mmap index file: ", err)
				}
				defer mmap.Unmap()

				p := bytes.NewBuffer(mmap)
				binary.Read(p, binary.BigEndian, &header)
				numberLessThanPrefix := int(header.Fanout[targetSHA1[0]-1])
				numberEqualAndLessThenPrefix := int(header.Fanout[targetSHA1[0]])
				p.Next(numberLessThanPrefix * int(unsafe.Sizeof(pio)))

				numberOfObjects := numberEqualAndLessThenPrefix - numberLessThanPrefix
				for i := 0; i < numberOfObjects; i++ {
					pio, _ := readIntoPackIndexObject(p)
					if testEq(pio.SHA1, targetSHA1) {
						log.Debugf("index: %s, pio: %s", index, pio)
						return
					}
				}
			}
			c <- 1
		}()
	}
	for i := 0; i < cap(c); i++ {
		<-c
	}
*/

/*
Split e.g. /foo/bar/meow.txt into (/foo/bar/meow, .txt)
or e.g meow.txt into (meow, .txt)
 */
func splitExt(path string) (root, ext string) {
	ext = filepath.Ext(path)
	root = path[:len(path)-len(ext)]
	return
}

// TODO copy/paste of GetBlobPackFile, refactor
func (apsi *ArqPackSetIndex) GetTreePackFile(abs *ArqBackupSet, ab *ArqBucket, targetSHA1 [20]byte) ([]byte, error) {
	indexes, err := apsi.ListTreeIndexes()
	if err != nil {
		log.Debugf("ArqPackSetIndex %s failed in GetPackFile to listIndexes: %s", err)
		return nil, err
	}
	var packIndexObjectResult *PackIndexObject
	var indexResult string
	for _, index := range indexes {
		indexContents, err := ioutil.ReadFile(index)
		if err != nil {
			log.Panicln(fmt.Sprintf("Could not read index file %s into memory. err: %s", index, err))
		}
		p := bytes.NewBuffer(indexContents)
		var header PackIndex
		binary.Read(p, binary.BigEndian, &header)
		numberLessThanPrefix := int(header.Fanout[targetSHA1[0]-1])
		numberEqualAndLessThenPrefix := int(header.Fanout[targetSHA1[0]])
		var pio PackIndexObject
		p.Next(numberLessThanPrefix * int(unsafe.Sizeof(pio)))

		numberOfObjects := numberEqualAndLessThenPrefix - numberLessThanPrefix
		for i := 0; i < numberOfObjects; i++ {
			pio, _ := readIntoPackIndexObject(p)
			if testEq(pio.SHA1, targetSHA1) {
				packIndexObjectResult = pio
				indexResult = index
				break
			}
		}
	}
	log.Debugf("GetTreePackFile packIndexObjectResult: %s, indexResult: %s", packIndexObjectResult, indexResult)
	if packIndexObjectResult == nil {
		err = errors.New(fmt.Sprintf("GetPackFile failed to find targetSHA1 %s",
			hex.EncodeToString(targetSHA1[:])))
		log.Debugf("%s", err)
		return nil, err
	}
	packName, _ := splitExt(filepath.Base(indexResult))
	log.Debugf("GetTreePackFile packName: %s", packName)

	pfo, err := GetObjectFromTreePackFile(abs, ab, packIndexObjectResult, packName)
	if err != nil {
		log.Debugf("GetPackFile failed to GetObjectFromTreePackFile: %s", err)
		return nil, err
	}

	decrypted, err := abs.BlobDecrypter.Decrypt(pfo.Data.Data)
	if err != nil {
		log.Debugf("GetPackFile failed to decrypt: %s", err)
		return nil, err
	}
	// Try to decompress, if fails then assume it was uncompressed to begin with
	var b bytes.Buffer
	r, err := gzip.NewReader(bytes.NewBuffer(decrypted))
	if err != nil {
		log.Debugf("GetPackFile decompression failed during NewReader, assume not compresed: ", err)
		return decrypted, nil
	}
	if _, err = io.Copy(&b, r); err != nil {
		log.Debugf("GetPackFile decompression failed during io.Copy, assume not compresed: ", err)
		return decrypted, nil
	}
	if err := r.Close(); err != nil {
		log.Debugf("GetPackFile decompression failed during reader Close, assume not compresed: ", err)
		return decrypted, nil
	}
	return b.Bytes(), nil
}

func (apsi *ArqPackSetIndex) GetBlobPackFile(abs *ArqBackupSet, ab *ArqBucket, targetSHA1 [20]byte) ([]byte, error) {
	indexes, err := apsi.ListBlobIndexes()
	if err != nil {
		log.Debugf("ArqPackSetIndex %s failed in GetBlobPackFile to listIndexes: %s", err)
		return nil, err
	}
	var packIndexObjectResult *PackIndexObject
	var indexResult string
	for _, index := range indexes {
		indexContents, err := ioutil.ReadFile(index)
		if err != nil {
			log.Panicln(fmt.Sprintf("Could not read index file %s into memory. err: %s", index, err))
		}
		p := bytes.NewBuffer(indexContents)
		var header PackIndex
		binary.Read(p, binary.BigEndian, &header)
		numberLessThanPrefix := int(header.Fanout[targetSHA1[0]-1])
		numberEqualAndLessThenPrefix := int(header.Fanout[targetSHA1[0]])
		var pio PackIndexObject
		p.Next(numberLessThanPrefix * int(unsafe.Sizeof(pio)))

		numberOfObjects := numberEqualAndLessThenPrefix - numberLessThanPrefix
		for i := 0; i < numberOfObjects; i++ {
			pio, _ := readIntoPackIndexObject(p)
			if testEq(pio.SHA1, targetSHA1) {
				packIndexObjectResult = pio
				indexResult = index
				break
			}
		}
	}
	if packIndexObjectResult == nil {
		err = errors.New(fmt.Sprintf("GetBlobPackFile failed to find targetSHA1 %s",
			hex.EncodeToString(targetSHA1[:])))
		log.Debugf("%s", err)
		return nil, err
	}
	packName, _ := splitExt(filepath.Base(indexResult))

	pfo, err := GetObjectFromBlobPackFile(abs, ab, packIndexObjectResult, packName)
	if err != nil {
		log.Debugf("GetBlobPackFile failed to GetObjectFromBlobPackFile: %s", err)
		return nil, err
	}
	decrypted, err := abs.BlobDecrypter.Decrypt(pfo.Data.Data)
	if err != nil {
		log.Debugf("GetBlobPackFile failed to decrypt: %s", err)
		return nil, err
	}
	// Try to decompress, if fails then assume it was uncompressed to begin with
	var b bytes.Buffer
	r, err := gzip.NewReader(bytes.NewBuffer(decrypted))
	if err != nil {
		log.Debugf("GetBlobPackFile decompression failed during NewReader, assume not compresed: ", err)
		return decrypted, nil
	}
	if _, err = io.Copy(&b, r); err != nil {
		log.Debugf("GetBlobPackFile decompression failed during io.Copy, assume not compresed: ", err)
		return decrypted, nil
	}
	if err := r.Close(); err != nil {
		log.Debugf("GetBlobPackFile decompression failed during reader Close, assume not compresed: ", err)
		return decrypted, nil
	}
	return b.Bytes(), nil
}

func (apsi *ArqPackSetIndex) ListTreeIndexes() ([]string, error) {
	rootDir := path.Join(apsi.CacheDirectory,
		GetPathToBucketPackSetTrees(apsi.ArqBackupSet, apsi.ArqBucket))
	return apsi.listIndexes(rootDir)
}

func (apsi *ArqPackSetIndex) ListBlobIndexes() ([]string, error) {
	rootDir := path.Join(apsi.CacheDirectory,
		GetPathToBucketPackSetBlobs(apsi.ArqBackupSet, apsi.ArqBucket))
	return apsi.listIndexes(rootDir)
}

func (apsi *ArqPackSetIndex) listIndexes(rootDir string) ([]string, error) {
	pattern := fmt.Sprintf("%s/*.index", rootDir)
	indexes, err := filepath.Glob(pattern)
	if err != nil {
		log.Debugln("listIndexes: failed to list indexes: ", err)
		return nil, err
	}
	return indexes, nil
}

type PackFileObject struct {
	Mimetype *arq_types.String
	Name     *arq_types.String
	Data     *arq_types.String
}

func NewPackFileObject(buf []byte) (*PackFileObject, error) {
	var err error
	pfo := PackFileObject{}
	p := bytes.NewBuffer(buf)
	if pfo.Mimetype, err = arq_types.ReadString(p); err != nil {
		log.Debugf("GetObjectFromTreePackFile failed during Mimetype parsing: %s", err)
		return nil, err
	}
	if pfo.Name, err = arq_types.ReadString(p); err != nil {
		log.Debugf("GetObjectFromTreePackFile failed during Name parsing: %s", err)
		return nil, err
	}
	var dataLength uint64
	binary.Read(p, binary.BigEndian, &dataLength)
	pfo.Data = &arq_types.String{Data: p.Next(int(dataLength))}
	if len(pfo.Data.Data) != int(dataLength) {
		log.Debugf("GetObjectFromTreePackFile expected %d bytes but only got %d", dataLength, len(pfo.Data.Data))
		return &pfo, errors.New("GetObjectFromTreePackFile didn't get enough bytes")
	}
	return &pfo, nil
}

func GetObjectFromTreePackFile(abs *ArqBackupSet, ab *ArqBucket, pio *PackIndexObject, packName string) (*PackFileObject, error) {
	key := path.Join(abs.UUID, "packsets",
		fmt.Sprintf("%s-trees", ab.UUID), fmt.Sprintf("%s.pack", packName))
	return GetObjectFromPackFile(key, abs, ab, pio, packName)
}

func GetObjectFromBlobPackFile(abs *ArqBackupSet, ab *ArqBucket, pio *PackIndexObject, packName string) (*PackFileObject, error) {
	key := path.Join(abs.UUID, "packsets",
		fmt.Sprintf("%s-blobs", ab.UUID), fmt.Sprintf("%s.pack", packName))
	return GetObjectFromPackFile(key, abs, ab, pio, packName)
}

func GetObjectFromPackFile(key string, abs *ArqBackupSet, ab *ArqBucket, pio *PackIndexObject, packName string) (*PackFileObject, error) {
	packFilepath, err := abs.Connection.CachedGet(key)
	if err != nil {
		log.Debugf("GetObjectFromPackFile failed first time to get key %s: %s", key, err)
	}
	isValid, err := IsValidPackFile(packFilepath)
	if (!isValid) {
		log.Debugf("GetObjectFromPackFile invalid pack file %s first time, will retry. err: %s", packFilepath, err)
		if err := os.Remove(packFilepath); err != nil {
			log.Debugf("GetObjectFromPackFile failed to delete pack file %s after detecting corruption. err: ", packFilepath, err)
			return nil, err
		}
		packFilepath, err := abs.Connection.CachedGet(key)
		if err != nil {
			log.Debugf("GetObjectFromPackFile failed second time to get key %s: %s", key, err)
		}
		isValid, err := IsValidPackFile(packFilepath)
		if (!isValid) {
			log.Debugf("GetObjectFromPackFile invalid pack file %s second time, will not retry. err: %s", packFilepath, err)
			return nil, err
		}
	}

	file, err := os.OpenFile(packFilepath, os.O_RDONLY, 0644)
	if err != nil {
		log.Debugf("GetObjectFromTreePackFile some error opening pack file %s: %s",
			packFilepath, err)
		return nil, err
	}
	defer file.Close()
	_, err = file.Seek(int64(pio.Offset), 0)
	if err != nil {
		log.Debugf("GetObjectFromTreePackFile Some error seeking %s for pio %s: %s",
			packFilepath, pio, err)
		return nil, err
	}

	// TODO In the pack index, the "data length" of an object only corresponds
	// to the size of the data in the pack file itself. This length ignores the
	// size of the mimetype (string) and name (string). Since these strings
	// are variable length it's not possible to know how much of the pack file
	// we need to load in order to get the object.
	//
	// I think the ideal solution is to mmap the file, but I've been getting
	// segmentation faults, possibly because I'm trying to use an offset which
	// isn't a multiple of the page size. So in the mean time just load the
	// whole file into memory...not ideal but since it's just a pack file
	// should be <= 10MB.
	//
	// https://github.com/edsrzf/mmap-go/blob/master/mmap.go#L53
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		log.Debugf("GetObjectFromTreePackFile some error reading: %s for pio %s: %s",
			packFilepath, pio, err)
		return nil, err
	}

	pfo, err := NewPackFileObject(buf)
	if err != nil {
		log.Debugf("GetObjectFromTreePackFile failed during NewPackFileObject: %s", err)
		return pfo, err
	}
	return pfo, nil
}
