package arq_types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type Tree struct {
	Header *Header

	// Only present for Tree v12 or later
	XattrsAreCompressed *Boolean

	// Only present for Tree v12 or later
	AclIsCompressed *Boolean

	XattrsBlobKey       *BlobKey
	XattrsSize          uint64
	AclBlobKey          *BlobKey
	Uid                 int32
	Gid                 int32
	Mode                int32
	MtimeSec            int64
	MtimeNsec           int64
	Flags               int64
	FinderFlags         int32
	ExtendedFinderFlags int32
	StDev               int32
	StIno               int32
	StNlink             uint32
	StRdev              int32
	CtimeSec            int64
	CtimeNsec           int64
	StBlocks            int64
	StBlksize           uint64

	// Only present for Tree v11 to v16
	AggregateSizeOnDisk uint64

	// Only present for Tree v15 or later
	CreateTimeSec int64

	// Only present for Tree v15 or later
	CreateTimeNsec int64

	// Only present for Tree v18 or later
	MissingNodes []*String

	Nodes []*TreeNode
}

func (c Tree) String() string {
	return fmt.Sprintf("{Tree: Header=%s, XattrsAreCompressed=%s, "+
		"AclIsCompressed=%s, XattrsBlobKey=%s, XattrsSize=%d, "+
		"AclBlobKey=%s, Uid=%d, Gid=%d, Mode=%d, MtimeSec=%d, MtimeNsec=%d}",
		c.Header, c.XattrsAreCompressed, c.AclIsCompressed, c.XattrsBlobKey,
		c.XattrsSize, c.AclBlobKey, c.Uid, c.Gid, c.Mode, c.MtimeSec,
		c.MtimeNsec)
}

func ReadTree(p *bytes.Buffer) (tree *Tree, err error) {
	tree = &Tree{}
	if tree.Header, err = ReadHeader(p); err != nil {
		err = errors.New(fmt.Sprintf("ReadTree header couldn't be parsed: %s", err))
		return
	}
	if tree.Header.Version >= 12 {
		if tree.XattrsAreCompressed, err = ReadBoolean(p); err != nil {
			err = errors.New(fmt.Sprintf("ReadTree failed during XattrsAreCompressed parsing: %s", err))
			return
		}
		if tree.AclIsCompressed, err = ReadBoolean(p); err != nil {
			err = errors.New(fmt.Sprintf("ReadTree failed during AclIsCompressed parsing: %s", err))
			return
		}
	}
	if tree.XattrsBlobKey, err = ReadBlobKey(p, tree.Header, true); err != nil {
		err = errors.New(fmt.Sprintf("ReadTree XattrsBlobKey couldn't be parsed: %s", err))
		return
	}
	binary.Read(p, binary.BigEndian, &tree.XattrsSize)
	if tree.AclBlobKey, err = ReadBlobKey(p, tree.Header, true); err != nil {
		err = errors.New(fmt.Sprintf("ReadTree AclBlobKey couldn't be parsed: %s", err))
		return
	}
	binary.Read(p, binary.BigEndian, &tree.Uid)
	binary.Read(p, binary.BigEndian, &tree.Gid)
	binary.Read(p, binary.BigEndian, &tree.Mode)
	binary.Read(p, binary.BigEndian, &tree.MtimeSec)
	binary.Read(p, binary.BigEndian, &tree.MtimeNsec)
	return
}
