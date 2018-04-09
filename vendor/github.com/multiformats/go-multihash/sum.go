package multihash

import (
	"crypto/sha1"
	"crypto/sha512"
	"errors"
	"fmt"

	keccak "github.com/gxed/hashland/keccakpg"
	blake2b "github.com/minio/blake2b-simd"
	sha256 "github.com/minio/sha256-simd"
	murmur3 "github.com/spaolacci/murmur3"
	blake2s "golang.org/x/crypto/blake2s"
	sha3 "golang.org/x/crypto/sha3"
)

// ErrSumNotSupported is returned when the Sum function code is not implemented
var ErrSumNotSupported = errors.New("Function not implemented. Complain to lib maintainer.")

// Sum obtains the cryptographic sum of a given buffer. The length parameter
// indicates the length of the resulting digest and passing a negative value
// use default length values for the selected hash function.
func Sum(data []byte, code uint64, length int) (Multihash, error) {
	m := Multihash{}
	err := error(nil)
	if !ValidCode(code) {
		return m, fmt.Errorf("invalid multihash code %d", code)
	}

	if length < 0 {
		var ok bool
		length, ok = DefaultLengths[code]
		if !ok {
			return m, fmt.Errorf("no default length for code %d", code)
		}
	}

	var d []byte
	switch {
	case isBlake2s(code):
		olen := code - BLAKE2S_MIN + 1
		switch olen {
		case 32:
			out := blake2s.Sum256(data)
			d = out[:]
		default:
			return nil, fmt.Errorf("unsupported length for blake2s: %d", olen)
		}
	case isBlake2b(code):
		olen := uint8(code - BLAKE2B_MIN + 1)
		d = sumBlake2b(olen, data)
	default:
		switch code {
		case ID:
			d = sumID(data)
		case SHA1:
			d = sumSHA1(data)
		case SHA2_256:
			d = sumSHA256(data)
		case SHA2_512:
			d = sumSHA512(data)
		case KECCAK_224:
			d = sumKeccak224(data)
		case KECCAK_256:
			d = sumKeccak256(data)
		case KECCAK_384:
			d = sumKeccak384(data)
		case KECCAK_512:
			d = sumKeccak512(data)
		case SHA3_224:
			d = sumSHA3_224(data)
		case SHA3_256:
			d = sumSHA3_256(data)
		case SHA3_384:
			d = sumSHA3_384(data)
		case SHA3_512:
			d = sumSHA3_512(data)
		case DBL_SHA2_256:
			d = sumSHA256(sumSHA256(data))
		case MURMUR3:
			d, err = sumMURMUR3(data)
		case SHAKE_128:
			d = sumSHAKE128(data)
		case SHAKE_256:
			d = sumSHAKE256(data)
		default:
			return m, ErrSumNotSupported
		}
	}
	if err != nil {
		return m, err
	}
	if length >= 0 {
		d = d[:length]
	}
	return Encode(d, code)
}

func isBlake2s(code uint64) bool {
	return code >= BLAKE2S_MIN && code <= BLAKE2S_MAX
}
func isBlake2b(code uint64) bool {
	return code >= BLAKE2B_MIN && code <= BLAKE2B_MAX
}

func sumBlake2b(size uint8, data []byte) []byte {
	hasher, err := blake2b.New(&blake2b.Config{Size: size})
	if err != nil {
		panic(err)
	}

	if _, err := hasher.Write(data); err != nil {
		panic(err)
	}

	return hasher.Sum(nil)[:]
}

func sumID(data []byte) []byte {
	return data
}

func sumSHA1(data []byte) []byte {
	a := sha1.Sum(data)
	return a[0:20]
}

func sumSHA256(data []byte) []byte {
	a := sha256.Sum256(data)
	return a[0:32]
}

func sumSHA512(data []byte) []byte {
	a := sha512.Sum512(data)
	return a[0:64]
}

func sumKeccak224(data []byte) []byte {
	h := keccak.New224()
	h.Write(data)
	return h.Sum(nil)
}

func sumKeccak256(data []byte) []byte {
	h := keccak.New256()
	h.Write(data)
	return h.Sum(nil)
}

func sumKeccak384(data []byte) []byte {
	h := keccak.New384()
	h.Write(data)
	return h.Sum(nil)
}

func sumKeccak512(data []byte) []byte {
	h := keccak.New512()
	h.Write(data)
	return h.Sum(nil)
}

func sumSHA3(data []byte) ([]byte, error) {
	h := sha3.New512()
	if _, err := h.Write(data); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func sumSHA3_512(data []byte) []byte {
	a := sha3.Sum512(data)
	return a[:]
}

func sumMURMUR3(data []byte) ([]byte, error) {
	number := murmur3.Sum32(data)
	bytes := make([]byte, 4)
	for i := range bytes {
		bytes[i] = byte(number & 0xff)
		number >>= 8
	}
	return bytes, nil
}

func sumSHAKE128(data []byte) []byte {
	bytes := make([]byte, 32)
	sha3.ShakeSum128(bytes, data)
	return bytes
}

func sumSHAKE256(data []byte) []byte {
	bytes := make([]byte, 64)
	sha3.ShakeSum256(bytes, data)
	return bytes
}

func sumSHA3_384(data []byte) []byte {
	a := sha3.Sum384(data)
	return a[:]
}

func sumSHA3_256(data []byte) []byte {
	a := sha3.Sum256(data)
	return a[:]
}

func sumSHA3_224(data []byte) []byte {
	a := sha3.Sum224(data)
	return a[:]
}
