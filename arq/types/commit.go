package arq_types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
)

const (
	READ_IS_COMPRESSED        = true
	DO_NOT_READ_IS_COMPRESSED = false
)

type Commit struct {
	Header        *Header
	Author        *String
	Comment       *String
	ParentCommits []*BlobKey
	TreeBlobKey   *BlobKey
	Location      *String

	// only present for Commit v7 or older, never used
	MergeCommonAncestorSHA1 *String

	// only present for Commit v4 to v7
	IsMergeCommonAncestorEncryptionKeyStretched *Boolean

	CreationDate      *Date
	CommitFailedFiles []*CommitFailedFile

	// only present for Commit v8 or later
	HasMissingNodes *Boolean

	// only present for Commit v9 or later
	IsComplete *Boolean

	// a copy of the XML file as described in s3_data_format.txt
	ConfigPlistXML *Data
}

func (c Commit) String() string {
	return fmt.Sprintf("{Commit: Header=%s, Author=%s, Comment=%s, "+
		"ParentCommits=%s, TreeBlobKey=%s, Location=%s, "+
		"MergeCommonAncestorSHA1=%s, "+
		"IsMergeCommonAncestorEncryptionKeyStretched=%s, "+
		"CreationDate=%s, CommitFailedFiles=%s, HasMissingNodes=%s, "+
		"IsComplete=%s}",
		c.Header, c.Author, c.Comment, c.ParentCommits, c.TreeBlobKey,
		c.Location, c.MergeCommonAncestorSHA1,
		c.IsMergeCommonAncestorEncryptionKeyStretched, c.CreationDate,
		c.CommitFailedFiles, c.HasMissingNodes, c.IsComplete)
}

func ReadCommit(p *bytes.Buffer) (commit *Commit, err error) {
	var err2 error
	commit = &Commit{}
	if commit.Header, err = ReadHeader(p); err != nil {
		err = errors.New(fmt.Sprintf("ReadCommit header couldn't be parsed: %s", err))
		return
	}
	if commit.Author, err = ReadString(p); err != nil {
		err = errors.New(fmt.Sprintf("ReadCommit failed during Author parsing: %s", err))
		return
	}
	if commit.Comment, err = ReadString(p); err != nil {
		err = errors.New(fmt.Sprintf("ReadCommit failed during Comment parsing: %s", err))
		return
	}
	var i, numParentCommits uint64
	binary.Read(p, binary.BigEndian, &numParentCommits)
	commit.ParentCommits = make([]*BlobKey, 0)
	for i = 0; i < numParentCommits; i++ {
		var parentCommit *BlobKey
		parentCommit, err = ReadBlobKey(p, commit.Header, DO_NOT_READ_IS_COMPRESSED)
		if err != nil {
			log.Printf("Failed to ReadBlobKey for commit %s: %s", commit, err)
			return
		}
		commit.ParentCommits = append(commit.ParentCommits, parentCommit)
	}
	commit.TreeBlobKey, err = ReadBlobKey(p, commit.Header, READ_IS_COMPRESSED)
	if err != nil {
		log.Printf("ReadCommit failed to read TreeBlobKey %s", err)
		return
	}
	if commit.Location, err2 = ReadString(p); err2 != nil {
		err = errors.New(fmt.Sprintf("ReadCommit failed during Location parsing: %s", err2))
		log.Printf("%s", err)
		return
	}
	if commit.Header.Version < 8 {
		if commit.MergeCommonAncestorSHA1, err2 = ReadString(p); err2 != nil {
			err = errors.New(fmt.Sprintf("ReadCommit failed during MergeCommonAncestorSHA1 parsing: %s", err2))
			log.Printf("%s", err)
			return
		}
		if commit.Header.Version >= 4 {
			if commit.IsMergeCommonAncestorEncryptionKeyStretched, err2 = ReadBoolean(p); err2 != nil {
				err = errors.New(fmt.Sprintf("ReadBlobKey failed during IsMergeCommonAncestorEncryptionKeyStretched parsing: %s", err2))
				return
			}
		}
	}
	if commit.CreationDate, err = ReadDate(p); err != nil {
		log.Printf("ReadCommit failed to read CreationDate %s", err)
		return
	}
	if commit.Header.Version >= 3 {
		var i, numFailedFiles uint64
		binary.Read(p, binary.BigEndian, &numFailedFiles)
		commit.CommitFailedFiles = make([]*CommitFailedFile, 0)
		for i = 0; i < numFailedFiles; i++ {
			var commitFailedFile *CommitFailedFile
			commitFailedFile, err = ReadCommitFailedFile(p)
			if err != nil {
				log.Printf("Failed to ReadCommitFailedFile for commit %s: %s", commit, err)
				return
			}
			commit.CommitFailedFiles = append(commit.CommitFailedFiles,
				commitFailedFile)
		}
	}
	if commit.Header.Version >= 8 {
		if commit.HasMissingNodes, err2 = ReadBoolean(p); err2 != nil {
			err = errors.New(fmt.Sprintf("ReadCommit failed during HasMissingNodes parsing: %s", err2))
			log.Printf("%s", err)
			return
		}
	}
	if commit.Header.Version >= 9 {
		if commit.IsComplete, err2 = ReadBoolean(p); err2 != nil {
			err = errors.New(fmt.Sprintf("ReadCommit failed during IsComplete parsing: %s", err2))
			log.Printf("%s", err)
			return
		}
	}
	if commit.Header.Version >= 5 {
		if commit.ConfigPlistXML, err2 = ReadData(p); err2 != nil {
			err = errors.New(fmt.Sprintf("ReadCommit failed during ConfigPlistXML parsing: %s", err2))
			log.Printf("%s", err)
			return
		}
	}
	return
}
