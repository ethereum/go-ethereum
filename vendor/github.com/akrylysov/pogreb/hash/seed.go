package hash

import (
	"crypto/rand"
	"encoding/binary"
)

// RandSeed generates a random hash seed.
func RandSeed() (uint32, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}
