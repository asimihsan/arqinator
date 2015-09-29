package arq_types

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

const (
	BLOB_TYPE_TREE       = iota
	BLOB_TYPE_COMMIT     = iota
	BLOB_TYPE_X_ATTR_SET = iota
)

type Header struct {
	Data    []byte
	Type    int
	Version int
}

func (h Header) String() string {
	return fmt.Sprintf("{Header: Type=%d, Version=%d, Data=%s}",
		h.Type, h.Version, h.Data)
}

func ReadHeader(p *bytes.Buffer) (header *Header, err error) {
	header = &Header{}
	header.Data = p.Next(10)
	var version []byte
	if bytes.HasPrefix(header.Data, []byte("CommitV")) {
		header.Type = BLOB_TYPE_COMMIT
		version = bytes.TrimPrefix(header.Data, []byte("CommitV"))
	} else if bytes.HasPrefix(header.Data, []byte("TreeV")) {
		header.Type = BLOB_TYPE_TREE
		version = bytes.TrimPrefix(header.Data, []byte("TreeV"))
	} else if bytes.HasPrefix(header.Data, []byte("XAttrSetV")) {
		header.Type = BLOB_TYPE_X_ATTR_SET
		version = bytes.TrimPrefix(header.Data, []byte("XAttrSetV"))
	} else {
		err = errors.New(fmt.Sprintf("ReadHeader header %s has unknown type", header.Data))
		return
	}
	if header.Version, err = strconv.Atoi(string(version)); err != nil {
		err = errors.New(fmt.Sprintf("ReadHeader header %s has non-integer version", header.Data))
		return
	}
	return
}
