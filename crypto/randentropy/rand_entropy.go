package randentropy

import (
	crand "crypto/rand"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"io"
	"os"
	"strings"
	"time"
)

var Reader io.Reader = &randEntropy{}

type randEntropy struct {
}

func (*randEntropy) Read(bytes []byte) (n int, err error) {
	readBytes := GetEntropyMixed(len(bytes))
	copy(bytes, readBytes)
	return len(bytes), nil
}

// TODO: copied from crypto.go , move to sha3 package?
func Sha3(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

// TODO: verify. this needs to be audited
// we start with crypt/rand, then XOR in additional entropy from OS
func GetEntropyMixed(n int) []byte {
	startTime := time.Now().UnixNano()
	// for each source, we take SHA3 of the source and use it as seed to math/rand
	// then read bytes from it and XOR them onto the bytes read from crypto/rand
	mainBuff := GetEntropyCSPRNG(n)
	// 1. OS entropy sources
	startTimeBytes := make([]byte, 32)
	binary.PutVarint(startTimeBytes, startTime)
	startTimeHash := Sha3(startTimeBytes)
	mixBytes(mainBuff, startTimeHash)

	pid := os.Getpid()
	pidBytes := make([]byte, 32)
	binary.PutUvarint(pidBytes, uint64(pid))
	pidHash := Sha3(pidBytes)
	mixBytes(mainBuff, pidHash)

	osEnv := os.Environ()
	osEnvBytes := []byte(strings.Join(osEnv, ""))
	osEnvHash := Sha3(osEnvBytes)
	mixBytes(mainBuff, osEnvHash)

	// not all OS have hostname in env variables
	osHostName, err := os.Hostname()
	if err != nil {
		osHostNameBytes := []byte(osHostName)
		osHostNameHash := Sha3(osHostNameBytes)
		mixBytes(mainBuff, osHostNameHash)
	}
	return mainBuff
}

func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(crand.Reader, mainBuff)
	if err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return mainBuff
}

func mixBytes(buff []byte, mixBuff []byte) []byte {
	bytesToMix := len(buff)
	if bytesToMix > 32 {
		bytesToMix = 32
	}
	for i := 0; i < bytesToMix; i++ {
		buff[i] ^= mixBuff[i]
	}
	return buff
}
