/*
arqinator: arq/types/commit_failed_file.go
Implements an Arq CommitFailedFile.

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

package arq_types

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
)

// Only present for Commit v3 or later
type CommitFailedFile struct {
	RelativePath *String
	ErrorMessage *String
}

func (cff CommitFailedFile) String() string {
	return fmt.Sprintf("{CommitFailedFile: RelativePath=%s, ErrorMessage=%s}",
		cff.RelativePath, cff.ErrorMessage)
}

func ReadCommitFailedFile(p *bytes.Buffer) (commitFailedFile *CommitFailedFile, err error) {
	var (
		err2 error
	)
	commitFailedFile = &CommitFailedFile{}
	if commitFailedFile.RelativePath, err2 = ReadString(p); err2 != nil {
		err = errors.New(fmt.Sprintf("ReadCommitFailedFile failed during RelativePath parsing: %s", err2))
		log.Debugf("%s", err)
		return
	}
	if commitFailedFile.ErrorMessage, err2 = ReadString(p); err2 != nil {
		err = errors.New(fmt.Sprintf("ReadCommitFailedFile failed during ErrorMessage parsing: %s", err2))
		log.Debugf("%s", err)
		return
	}
	return
}
