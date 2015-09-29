package arq_types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
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
		log.Printf("ReadData failed to read byte: %s", err)
		return
	}
	if isNull == 1 {
		var epochMs uint64
		err = binary.Read(p, binary.BigEndian, &epochMs)
		if err != nil {
			log.Printf("ReadData failed during read of epochMs %d: %s",
				epochMs, err)
			return
		}
		date.IsPresent = true
		date.Data = time.Unix(0, int64(epochMs*uint64(time.Millisecond)))
		return
	}
	return
}
