package arq_types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

type Data struct {
	IsPresent bool
	Data      []byte
}

func (d Data) String() string {
	if !d.IsPresent {
		return "<nil>"
	}
	return fmt.Sprintf("%s", d.Data)
}

func ReadData(p *bytes.Buffer) (data *Data, err error) {
	data = &Data{}
	var length uint64
	err = binary.Read(p, binary.BigEndian, &length)
	if err != nil {
		log.Printf("ReadData failed during read of length %d: %s",
			length, err)
		return
	}
	data.IsPresent = true
	data.Data = p.Next(int(length))
	return
}
