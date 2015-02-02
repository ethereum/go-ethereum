package secp256k1

import (
	crand "crypto/rand"
	"io"
	mrand "math/rand"
	"os"
	"strings"
	"time"
)

/*
Note:

- On windows cryto/rand uses CrytoGenRandom which uses RC4 which is insecure
- Android random number generator is known to be insecure.
- Linux uses /dev/urandom , which is thought to be secure and uses entropy pool

Therefore the output is salted.
*/

//finalizer from MurmerHash3
func mmh3f(key uint64) uint64 {
	key ^= key >> 33
	key *= 0xff51afd7ed558ccd
	key ^= key >> 33
	key *= 0xc4ceb9fe1a85ec53
	key ^= key >> 33
	return key
}

//knuth hash
func knuth_hash(in []byte) uint64 {
	var acc uint64 = 3074457345618258791
	for i := 0; i < len(in); i++ {
		acc += uint64(in[i])
		acc *= 3074457345618258799
	}
	return acc
}

var _rand *mrand.Rand

func init() {
	var seed1 uint64 = mmh3f(uint64(time.Now().UnixNano()))
	var seed2 uint64 = knuth_hash([]byte(strings.Join(os.Environ(), "")))
	var seed3 uint64 = mmh3f(uint64(os.Getpid()))

	_rand = mrand.New(mrand.NewSource(int64(seed1 ^ seed2 ^ seed3)))
}

func saltByte(n int) []byte {
	buff := make([]byte, n)
	for i := 0; i < len(buff); i++ {
		var v uint64 = uint64(_rand.Int63())
		var b byte
		for j := 0; j < 8; j++ {
			b ^= byte(v & 0xff)
			v = v >> 8
		}
		buff[i] = b
	}
	return buff
}

//On Unix-like systems, Reader reads from /dev/urandom.
//On Windows systems, Reader uses the CryptGenRandom API.

//use entropy pool etc and cryptographic random number generator
//mix in time
//mix in mix in cpu cycle count
func RandByte(n int) []byte {
	buff := make([]byte, n)
	ret, err := io.ReadFull(crand.Reader, buff)
	if len(buff) != ret || err != nil {
		return nil
	}

	buff2 := saltByte(n)
	for i := 0; i < n; i++ {
		buff[i] ^= buff2[2]
	}
	return buff
}

/*
	On Unix-like systems, Reader reads from /dev/urandom.
	On Windows systems, Reader uses the CryptGenRandom API.
*/
func RandByteWeakCrypto(n int) []byte {
	buff := make([]byte, n)
	ret, err := io.ReadFull(crand.Reader, buff)
	if len(buff) != ret || err != nil {
		return nil
	}
	return buff
}
