/*
arqinator: arq/types/header.go
Implements Arq headers.

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

func getHeaderTypeAsString(headerType int) string {
	switch headerType {
	case BLOB_TYPE_TREE:
		return "BLOB_TYPE_TREE"
	case BLOB_TYPE_COMMIT:
		return "BLOB_TYPE_COMMIT"
	case BLOB_TYPE_X_ATTR_SET:
		return "BLOB_TYPE_X_ATTR_SET"
	default:
		return "<unknown>"
	}

}

func (h Header) String() string {
	return fmt.Sprintf("{Header: Type=%s, Version=%d, Data=%s}",
		getHeaderTypeAsString(h.Type), h.Version, h.Data)
}

// CommitV000
// TreeV000
// XAttrSetV000
func ReadHeader(p *bytes.Buffer) (header *Header, err error) {
	header = &Header{}
	prefix := p.Next(4)
	var version []byte
	if bytes.Equal(prefix, []byte("Comm")) {
		header.Data = append(prefix, p.Next(6)...)
		header.Type = BLOB_TYPE_COMMIT
		version = bytes.TrimPrefix(header.Data, []byte("CommitV"))
	} else if bytes.Equal(prefix, []byte("Tree")) {
		header.Data = append(prefix, p.Next(4)...)
		header.Type = BLOB_TYPE_TREE
		version = bytes.TrimPrefix(header.Data, []byte("TreeV"))
	} else if bytes.Equal(prefix, []byte("XAtt")) {
		header.Data = append(prefix, p.Next(8)...)
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
