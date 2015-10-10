/*
arqinator: arq/types/bucket.go
Implements an Arq Bucket, a particular folder backed up in a backup set.

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
	Object       connector.Object
	UUID         string
	LocalPath    string
	ArqBackupSet *ArqBackupSet
	HeadSHA1     [20]byte
}

func (ab ArqBucket) String() string {
	return fmt.Sprintf("{ArqBucket: UUID=%s, LocalPath=%s, S3Obj=%s, HeadSHA1=%s}",
		ab.UUID, ab.LocalPath, ab.Object, hex.EncodeToString(ab.HeadSHA1[:]))
}

func (ab *ArqBucket) parsePlist() error {
	filepath, err := ab.ArqBackupSet.Connection.CachedGet(ab.Object.GetPath())
	if err != nil {
		log.Debugln("Failed during NewArqBucket for s3Obj: ", ab.Object)
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
	key := path.Join(ab.ArqBackupSet.UUID, "bucketdata", ab.UUID,
		"refs", "heads", "master")
	filepath, err := ab.ArqBackupSet.Connection.Get(key)
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

func NewArqBucket(object connector.Object, abs *ArqBackupSet) (*ArqBucket, error) {
	bucket := ArqBucket{Object: object, ArqBackupSet: abs}
	bucket.UUID = path.Base(object.GetPath())
	err := bucket.parsePlist()
	if err != nil {
		err2 := errors.New(fmt.Sprintf("Failed during NewArqBucket: %s", err))
		log.Debugf("%s", err2)
		return nil, err2
	}
	bucket.updateHeadSHA1()
	return &bucket, nil
}
