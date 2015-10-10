/*
arqinator: arq/types/tree_node.go
Implements an Arq TreeNode, a type of Node specific to Trees.

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
