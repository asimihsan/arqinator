package arq_types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
)

type String struct {
	Data []byte
}

func (s String) String() string {
	if s.Data == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s", s.Data)
}

func (s *String) Equal(o string) bool {
	return bytes.Equal(s.Data, []byte(o))
}

func ReadString(p *bytes.Buffer) (*String, error) {
	isNull, err := p.ReadByte()
	if err != nil {
		log.Debugf("ReadString failed to read byte: %s", err)
		return nil, err
	}
	if isNull == 1 {
		var length uint64
		err = binary.Read(p, binary.BigEndian, &length)
		if err != nil {
			log.Debugf("ReadString failed during read of length %d: %s",
				length, err)
			return nil, err
		}
		return &String{Data: p.Next(int(length))}, nil
	}
	return nil, nil
}

func ReadStringAsSHA1(p *bytes.Buffer) (*[20]byte, error) {
	var (
		result [20]byte
	)
	data1, err := ReadString(p)
	if err != nil {
		err = errors.New(fmt.Sprintf("ReadStringAsSHA1 failed during SHA1 parsing: %s", err))
		log.Debugf("%s", err)
		return nil, err
	}
	if data1 == nil {
		return nil, nil
	}
	data2, err := hex.DecodeString(string(data1.Data))
	if err != nil {
		err = errors.New(fmt.Sprintf("ReadStringAsSHA1 failed to hex decode %s hex: %s",
			data1, err))
		log.Debugf("%s", err)
		return nil, err
	}
	copy(result[:], data2)
	return &result, nil
}
