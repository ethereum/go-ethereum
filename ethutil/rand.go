package ethutil

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

func randomUint64(r io.Reader) (uint64, error) {
	b := make([]byte, 8)
	n, err := r.Read(b)
	if n != len(b) {
		return 0, io.ErrShortBuffer
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

// RandomUint64 returns a cryptographically random uint64 value.
func RandomUint64() (uint64, error) {
	return randomUint64(rand.Reader)
}
