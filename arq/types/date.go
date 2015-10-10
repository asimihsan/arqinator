/*
arqinator: arq/types/date.go
Implements an Arq Date.

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
	"encoding/binary"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"time"
)

type Date struct {
	IsPresent bool
	Data      time.Time
}

func (d Date) String() string {
	if !d.IsPresent {
		return "<nil>"
	}
	return fmt.Sprintf("%s", d.Data)
}

func ReadDate(p *bytes.Buffer) (date *Date, err error) {
	date = &Date{}
	isNull, err := p.ReadByte()
	if err != nil {
		log.Debugf("ReadData failed to read byte: %s", err)
		return
	}
	if isNull == 1 {
		var epochMs uint64
		err = binary.Read(p, binary.BigEndian, &epochMs)
		if err != nil {
			log.Debugf("ReadData failed during read of epochMs %d: %s",
				epochMs, err)
			return
		}
		date.IsPresent = true
		date.Data = time.Unix(0, int64(epochMs*uint64(time.Millisecond)))
		return
	}
	return
}
