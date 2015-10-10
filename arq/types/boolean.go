/*
arqinator: arq/types/boolean.go
Implements an Arq Boolean.

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
	"fmt"
	log "github.com/Sirupsen/logrus"
)

type Boolean struct {
	IsPresent bool
	Data      bool
}

func (b Boolean) String() string {
	if !b.IsPresent {
		return "<nil>"
	}
	return fmt.Sprintf("%t", b.Data)
}

func (b *Boolean) IsTrue() bool {
	return b.IsPresent && b.Data
}

func ReadBoolean(p *bytes.Buffer) (boolean *Boolean, err error) {
	boolean = &Boolean{}
	isTrue, err := p.ReadByte()
	if err != nil {
		log.Debugf("ReadString failed to read byte: %s", err)
		return
	}
	boolean.IsPresent = true
	boolean.Data = isTrue == 1
	return
}
