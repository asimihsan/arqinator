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
