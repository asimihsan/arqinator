package arq_types

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
)

// Only present for Commit v3 or later
type TreeNode struct {
	Filename *String
	//Node *Node
}

func (n TreeNode) String() string {
	return fmt.Sprintf("{TreeNode: Filename=%s}",
		n.Filename)
}

func ReadTreeNode(p *bytes.Buffer) (treeNode *TreeNode, err error) {
	var (
		err2 error
	)
	treeNode = &TreeNode{}
	if treeNode.Filename, err2 = ReadString(p); err2 != nil {
		err = errors.New(fmt.Sprintf("ReadTreeNode failed during Filename parsing: %s", err2))
		log.Debugf("%s", err)
		return
	}
	return
}
