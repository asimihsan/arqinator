package arq

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"unsafe"

	"github.com/edsrzf/mmap-go"

	"github.com/asimihsan/arqinator/arq/types"
)

type ArqPackSetIndex struct {
	CacheDirectory string
	ArqBackupSet   *ArqBackupSet
	ArqBucket      *ArqBucket
}

func GetPathToBucketPackSetTrees(abs *ArqBackupSet, ab *ArqBucket) string {
	return path.Join(abs.Uuid, "packsets", fmt.Sprintf("%s-trees", ab.UUID))
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
		log.Printf("Failed during readIntoPackIndexObject: %s", err)
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
						log.Printf("index: %s, pio: %s", index, pio)
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

func splitExt(path string) (root, ext string) {
	ext = filepath.Ext(path)
	root = path[:len(path)-len(ext)]
	return
}

func (apsi *ArqPackSetIndex) GetPackFile(abs *ArqBackupSet, ab *ArqBucket, targetSHA1 [20]byte) ([]byte, error) {
	indexes, err := apsi.listIndexes()
	if err != nil {
		log.Printf("ArqPackSetIndex %s failed in GetPackFile to listIndexes: %s", err)
		return nil, err
	}
	var packIndexObjectResult *PackIndexObject
	var indexResult string
	for _, index := range indexes {
		func() {
			file, err := os.OpenFile(index, os.O_RDONLY, 0644)
			if err != nil {
				log.Panicln("Couldn't open index file: ", err)
			}
			defer file.Close()
			mmap, err := mmap.MapRegion(file, -1, mmap.RDONLY, 0, 0)
			if err != nil {
				log.Panicln("Failed to mmap index file: ", err)
			}
			defer mmap.Unmap()

			p := bytes.NewBuffer(mmap)
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
		}()
	}
	if packIndexObjectResult == nil {
		err = errors.New(fmt.Sprintf("GetPackFile failed to find targetSHA1 %s",
			targetSHA1))
		log.Printf("%s", err)
		return nil, err
	}
	packName, _ := splitExt(path.Base(indexResult))
	pfo, err := GetObjectFromTreePackFile(abs, ab, packIndexObjectResult, packName)
	if err != nil {
		log.Printf("GetPackFile failed to GetObjectFromTreePackFile: %s", err)
		return nil, err
	}
	decrypted := abs.BlobDecrypter.Decrypt(pfo.Data.Data)
	return decrypted, nil
}

func (apsi *ArqPackSetIndex) listIndexes() ([]string, error) {
	root_dir := path.Join(apsi.CacheDirectory,
		GetPathToBucketPackSetTrees(apsi.ArqBackupSet, apsi.ArqBucket))
	pattern := fmt.Sprintf("%s/*.index", root_dir)
	indexes, err := filepath.Glob(pattern)
	if err != nil {
		log.Println("GetPackFile failed to list indexes: ", err)
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
		log.Printf("GetObjectFromTreePackFile failed during Mimetype parsing: %s", err)
		return nil, err
	}
	if pfo.Name, err = arq_types.ReadString(p); err != nil {
		log.Printf("GetObjectFromTreePackFile failed during Name parsing: %s", err)
		return nil, err
	}
	var dataLength uint64
	binary.Read(p, binary.BigEndian, &dataLength)
	pfo.Data = &arq_types.String{Data: p.Next(int(dataLength))}
	if len(pfo.Data.Data) != int(dataLength) {
		log.Printf("GetObjectFromTreePackFile expected %d bytes but only got %d", dataLength, len(pfo.Data.Data))
		return &pfo, errors.New("GetObjectFromTreePackFile didn't get enough bytes")
	}
	return &pfo, nil
}

func GetObjectFromTreePackFile(abs *ArqBackupSet, ab *ArqBucket, pio *PackIndexObject, packName string) (*PackFileObject, error) {
	key := path.Join(abs.Uuid, "packsets",
		fmt.Sprintf("%s-trees", ab.UUID), fmt.Sprintf("%s.pack", packName))
	packFilepath, err := abs.Connection.CachedGet(abs.S3BucketName, key)
	if err != nil {
		log.Printf("GetObjectFromTreePackFile failed to get key %s: %s", key, err)
		return nil, err
	}
	file, err := os.OpenFile(packFilepath, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("GetObjectFromTreePackFile some error opening pack file %s: %s",
			packFilepath, err)
		return nil, err
	}
	defer file.Close()
	_, err = file.Seek(int64(pio.Offset), 0)
	if err != nil {
		log.Printf("GetObjectFromTreePackFile Some error seeking %s for pio %s: %s",
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
		log.Printf("GetObjectFromTreePackFile some error reading: %s for pio %s: %s",
			packFilepath, pio, err)
		return nil, err
	}

	pfo, err := NewPackFileObject(buf)
	if err != nil {
		log.Printf("GetObjectFromTreePackFile failed during NewPackFileObject: %s", err)
		return pfo, err
	}
	return pfo, nil
}
