package arq

import (
	"bytes"
	"encoding/binary"
	"log"
)

func ReadString(p *bytes.Buffer, output *[]byte) error {
	isNull, err := p.ReadByte()
	if err != nil {
		log.Printf("ReadString failed to read byte: %s", err)
		return err
	}
	if isNull == 1 {
		var length uint64
		err = binary.Read(p, binary.BigEndian, &length)
		if err != nil {
			log.Printf("ReadString failed during read of length %d: %s",
				length, err)
			return err
		}
		*output = p.Next(int(length))
	}
	return nil
}
