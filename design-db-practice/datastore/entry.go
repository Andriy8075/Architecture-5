package datastore

import (
	"bufio"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type entry struct {
	key, value string
	hash       [20]byte
}

// 0           4    8     kl+8  kl+12     <-- offset
// (full size) (kl) (key) (vl)  (value)
// 4           4    ....  4     .....     <-- length

func (e *entry) Encode() []byte {
	kl, vl := len(e.key), len(e.value)
	e.hash = sha1.Sum([]byte(e.key + e.value))
	size := kl + vl + 12 + len(e.hash)
	res := make([]byte, size)
	binary.LittleEndian.PutUint32(res, uint32(size))
	binary.LittleEndian.PutUint32(res[4:], uint32(kl))
	copy(res[8:], e.key)
	binary.LittleEndian.PutUint32(res[kl+8:], uint32(vl))
	copy(res[kl+12:], e.value)
	copy(res[kl+12+vl:], e.hash[:])
	return res
}

func (e *entry) Decode(input []byte) {
	kl := int(binary.LittleEndian.Uint32(input[4:8]))
	vl := int(binary.LittleEndian.Uint32(input[kl+8 : kl+12]))
	e.key = string(input[8 : 8+kl])
	e.value = string(input[kl+12 : kl+12+vl])
	copy(e.hash[:], input[len(input)-20:])
}

func decodeString(v []byte) string {
	l := binary.LittleEndian.Uint32(v)
	buf := make([]byte, l)
	copy(buf, v[4:4+int(l)])
	return string(buf)
}

func (e *entry) DecodeFromReader(in *bufio.Reader) (int, error) {
	sizeBuf, err := in.Peek(4)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, err
		}
		return 0, fmt.Errorf("DecodeFromReader, cannot read size: %w", err)
	}
	buf := make([]byte, int(binary.LittleEndian.Uint32(sizeBuf)))
	n, err := in.Read(buf[:])
	if err != nil {
		return n, fmt.Errorf("DecodeFromReader, cannot read record: %w", err)
	}
	e.Decode(buf)
	return n, nil
}

func (e *entry) EncodeHash() [20]byte {
	return sha1.Sum([]byte(e.key + e.value))
}
