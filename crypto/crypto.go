package crypto

// https://code.google.com/p/rsc/source/browse/arq/crypto.go

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"golang.org/x/crypto/pbkdf2"
	"hash"
	log "github.com/Sirupsen/logrus"
	"errors"
)

type CryptoState struct {
	c  cipher.Block
	iv []byte
}

func NewCryptoState(password []byte, salt []byte) (*CryptoState, error) {
	const (
		PBKDF2_ITERATIONS = 1000
		AES_KEY_LEN_BYTES = 32
		AES_IV_LEN_BYTES  = 16
	)
	var err error
	state := CryptoState{}
	key1 := pbkdf2.Key(password, salt, PBKDF2_ITERATIONS,
		AES_KEY_LEN_BYTES+AES_IV_LEN_BYTES, sha1.New)
	var key2 []byte
	key2, state.iv = bytesToKey(sha1.New, salt, key1, PBKDF2_ITERATIONS,
		AES_KEY_LEN_BYTES, AES_IV_LEN_BYTES)
	if state.c, err = aes.NewCipher(key2); err != nil {
		log.Debugln("Failed to create aes cipher object in crypto NewState",
			err)
		return nil, err
	}
	return &state, nil
}

func (s *CryptoState) Decrypt(data []byte) ([]byte, error) {
	data = bytes.TrimPrefix(data, []byte("encrypted"))
	dec := cipher.NewCBCDecrypter(s.c, s.iv)
	if len(data)%aes.BlockSize != 0 {
		err := errors.New("Decrypt data length not multiple of AES block size")
		return nil, err
	}
	if len(data) == 0 {
		err := errors.New("Decrypt data length is zero, not expected")
		return nil, err
	}
	dec.CryptBlocks(data, data)
	//log.Debugf("% x\n", data)
	//log.Debugf("%s\n", data)

	// unpad
	{
		n := len(data)
		p := int(data[n-1])
		if p == 0 || p > aes.BlockSize {
			err := errors.New("Decrypt impossible padding, bad password?")
			return nil, err
		}
		for i := 0; i < p; i++ {
			if data[n-1-i] != byte(p) {
				err := errors.New("Decrypt bad padding, bad password?")
				return nil, err
			}
		}
		data = data[:n-p]
	}
	return data, nil
}

func bytesToKey(hf func() hash.Hash, salt, data []byte, iter int, keySize,
	ivSize int) (key, iv []byte) {
	h := hf()
	var d, dcat []byte
	sum := make([]byte, 0, h.Size())
	for len(dcat) < keySize+ivSize {
		// D_i = HASH^count(D_(i-1) || data || salt)
		h.Reset()
		h.Write(d)
		h.Write(data)
		h.Write(salt)
		sum = h.Sum(sum[:0])

		for j := 1; j < iter; j++ {
			h.Reset()
			h.Write(sum)
			sum = h.Sum(sum[:0])
		}

		d = append(d[:0], sum...)
		dcat = append(dcat, d...)
	}

	return dcat[:keySize], dcat[keySize : keySize+ivSize]
}
