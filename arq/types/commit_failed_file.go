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
