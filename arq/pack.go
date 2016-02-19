/*
arqinator: arq/pack.go

Implements functions common to pack files (both trees and blobs, both .index and .pack files).

Copyright 2016 Asim Ihsan

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
	"crypto/sha1"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"

	log "github.com/Sirupsen/logrus"
)

/**
Verify that a pack file is valid. All pack files end with a 20-byte SHA1 of the full contents of the file.
 */
func IsValidPackFile(cacheFilepath string) (bool, error) {
	fileInfo, err := os.Stat(cacheFilepath)
	if err != nil {
		// some error whilst reading the file from the file system.
		log.Debugln("IsValidPackFile: some error while stat-ing file: ", err)
		return false, err
	}
	size := fileInfo.Size()
	if (size <= 20) {
		// the file itself is <= 20 bytes big, so the file must be truncated
		msg := fmt.Sprintf("IsValidPackFile: file is too small, must be truncated. size: %s", fileInfo.Size())
		log.Debugln(msg)
		return false, errors.New(msg)
	}
	f, err := os.Open(cacheFilepath)
	if err != nil {
		log.Debugln("IsValidPackFile: some error while opening file: ", err)
		return false, err
	}
	defer f.Close()
	hasher := sha1.New()
	if _, err := io.CopyN(hasher, f, size - 20); err != nil {
		log.Debugln("IsValidPackFile: error whilst reading pack file: ", err)
		return false, err
	}
	calculatedHash := hasher.Sum(nil)
	expectedHash := make([]byte, 20)
	if _, err := io.ReadAtLeast(f, expectedHash, 20); err != nil {
		log.Debugln("IsValidPackFile: error whilst reading last 20 bytes of file: ", err)
		return false, err
	}

	result := subtle.ConstantTimeCompare(calculatedHash, expectedHash) == 1
	if !result {
		return false, errors.New("IsValidPackFile hash checksum does not match file contents")
	}
	return true, nil
}
