package randentropy

import (
	crand "crypto/rand"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"io"
)

var Reader io.Reader = &randEntropy{}

type randEntropy struct {
}

func (*randEntropy) Read(bytes []byte) (n int, err error) {
	readBytes := GetEntropyCSPRNG(len(bytes))
	copy(bytes, readBytes)
	return len(bytes), nil
}

// TODO: copied from crypto.go , move to sha3 package?
func Sha3(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(crand.Reader, mainBuff)
	if err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return mainBuff
}
