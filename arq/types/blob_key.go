package arq_types

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
)

const (
	STORAGE_TYPE_S3      = uint32(1)
	STORAGE_TYPE_GLACIER = uint32(2)
)

func (b BlobKey) String() string {
	return fmt.Sprintf("{BlobKey: SHA1=%s, IsEncryptionKeyStretched=%s, "+
		"IsCompressed=%s}",
		hex.EncodeToString(b.SHA1[:]), b.IsEncryptionKeyStretched,
		b.IsCompressed)
}

type BlobKey struct {
	SHA1 [20]byte

	// only present for Tree v14 or later or Commit v4 or later
	// applies for both trees and commits
	IsEncryptionKeyStretched *Boolean

	// only present for Commit v8 or later
	// only for a Tree BlobKey, not for ParentCommit BlobKey
	IsCompressed *Boolean
}

func ReadBlobKey(p *bytes.Buffer, h *Header, readIsCompressed bool) (blobKey *BlobKey, err error) {
	var (
		err2 error
	)

	blobKey = &BlobKey{}
	if blobKey.SHA1, err2 = ReadStringAsSHA1(p); err2 != nil {
		err = errors.New(fmt.Sprintf("ReadBlobKey failed to hex decode hex: %s", err2))
		log.Printf("%s", err)
		return
	}
	if (h.Type == BLOB_TYPE_TREE && h.Version >= 14) ||
		(h.Type == BLOB_TYPE_COMMIT && h.Version >= 4) {
		if blobKey.IsEncryptionKeyStretched, err2 = ReadBoolean(p); err2 != nil {
			err = errors.New(fmt.Sprintf("ReadBlobKey failed during IsEncryptionKeyStretched parsing: %s", err2))
			return
		}
	}
	if h.Type == BLOB_TYPE_COMMIT && h.Version >= 8 && readIsCompressed {
		if blobKey.IsCompressed, err2 = ReadBoolean(p); err2 != nil {
			err = errors.New(fmt.Sprintf("ReadBlobKey failed during IsCompressed parsing: %s", err2))
			return
		}
	}
	return
}
