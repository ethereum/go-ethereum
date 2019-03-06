package multihash

import (
	"crypto/sha1"
	"crypto/sha512"
	"errors"
	"fmt"

	keccak "github.com/gxed/hashland/keccakpg"
	murmur3 "github.com/gxed/hashland/murmur3"
	blake2b "github.com/minio/blake2b-simd"
	sha256 "github.com/minio/sha256-simd"
	blake2s "golang.org/x/crypto/blake2s"
	sha3 "golang.org/x/crypto/sha3"
)

// ErrSumNotSupported is returned when the Sum function code is not implemented
var ErrSumNotSupported = errors.New("Function not implemented. Complain to lib maintainer.")

// HashFunc is a hash function that hashes data into digest.
//
// The length is the size the digest will be truncated to. While the hash
// function isn't responsible for truncating the digest, it may want to error if
// the length is invalid for the hash function (e.g., truncation would make the
// hash useless).
type HashFunc func(data []byte, length int) (digest []byte, err error)

// funcTable maps multicodec values to hash functions.
var funcTable = make(map[uint64]HashFunc)

// Sum obtains the cryptographic sum of a given buffer. The length parameter
// indicates the length of the resulting digest and passing a negative value
// use default length values for the selected hash function.
func Sum(data []byte, code uint64, length int) (Multihash, error) {
	if !ValidCode(code) {
		return nil, fmt.Errorf("invalid multihash code %d", code)
	}

	if length < 0 {
		var ok bool
		length, ok = DefaultLengths[code]
		if !ok {
			return nil, fmt.Errorf("no default length for code %d", code)
		}
	}

	hashFunc, ok := funcTable[code]
	if !ok {
		return nil, ErrSumNotSupported
	}

	d, err := hashFunc(data, length)
	if err != nil {
		return nil, err
	}
	if length >= 0 {
		d = d[:length]
	}
	return Encode(d, code)
}

func sumBlake2s(data []byte, size int) ([]byte, error) {
	if size != 32 {
		return nil, fmt.Errorf("unsupported length for blake2s: %d", size)
	}
	d := blake2s.Sum256(data)
	return d[:], nil
}
func sumBlake2b(data []byte, size int) ([]byte, error) {
	hasher, err := blake2b.New(&blake2b.Config{Size: uint8(size)})
	if err != nil {
		return nil, err
	}

	if _, err := hasher.Write(data); err != nil {
		return nil, err
	}

	return hasher.Sum(nil)[:], nil
}

func sumID(data []byte, length int) ([]byte, error) {
	if length >= 0 && length != len(data) {
		return nil, fmt.Errorf("the length of the identity hash (%d) must be equal to the length of the data (%d)",
			length, len(data))

	}
	return data, nil
}

func sumSHA1(data []byte, length int) ([]byte, error) {
	a := sha1.Sum(data)
	return a[0:20], nil
}

func sumSHA256(data []byte, length int) ([]byte, error) {
	a := sha256.Sum256(data)
	return a[0:32], nil
}

func sumDoubleSHA256(data []byte, length int) ([]byte, error) {
	val, _ := sumSHA256(data, len(data))
	return sumSHA256(val, len(val))
}

func sumSHA512(data []byte, length int) ([]byte, error) {
	a := sha512.Sum512(data)
	return a[0:64], nil
}

func sumKeccak224(data []byte, length int) ([]byte, error) {
	h := keccak.New224()
	h.Write(data)
	return h.Sum(nil), nil
}

func sumKeccak256(data []byte, length int) ([]byte, error) {
	h := keccak.New256()
	h.Write(data)
	return h.Sum(nil), nil
}

func sumKeccak384(data []byte, length int) ([]byte, error) {
	h := keccak.New384()
	h.Write(data)
	return h.Sum(nil), nil
}

func sumKeccak512(data []byte, length int) ([]byte, error) {
	h := keccak.New512()
	h.Write(data)
	return h.Sum(nil), nil
}

func sumSHA3_512(data []byte, length int) ([]byte, error) {
	a := sha3.Sum512(data)
	return a[:], nil
}

func sumMURMUR3(data []byte, length int) ([]byte, error) {
	number := murmur3.Sum32(data)
	bytes := make([]byte, 4)
	for i := range bytes {
		bytes[i] = byte(number & 0xff)
		number >>= 8
	}
	return bytes, nil
}

func sumSHAKE128(data []byte, length int) ([]byte, error) {
	bytes := make([]byte, 32)
	sha3.ShakeSum128(bytes, data)
	return bytes, nil
}

func sumSHAKE256(data []byte, length int) ([]byte, error) {
	bytes := make([]byte, 64)
	sha3.ShakeSum256(bytes, data)
	return bytes, nil
}

func sumSHA3_384(data []byte, length int) ([]byte, error) {
	a := sha3.Sum384(data)
	return a[:], nil
}

func sumSHA3_256(data []byte, length int) ([]byte, error) {
	a := sha3.Sum256(data)
	return a[:], nil
}

func sumSHA3_224(data []byte, length int) ([]byte, error) {
	a := sha3.Sum224(data)
	return a[:], nil
}

func registerStdlibHashFuncs() {
	RegisterHashFunc(ID, sumID)
	RegisterHashFunc(SHA1, sumSHA1)
	RegisterHashFunc(SHA2_512, sumSHA512)
}

func registerNonStdlibHashFuncs() {
	RegisterHashFunc(SHA2_256, sumSHA256)
	RegisterHashFunc(DBL_SHA2_256, sumDoubleSHA256)

	RegisterHashFunc(KECCAK_224, sumKeccak224)
	RegisterHashFunc(KECCAK_256, sumKeccak256)
	RegisterHashFunc(KECCAK_384, sumKeccak384)
	RegisterHashFunc(KECCAK_512, sumKeccak512)

	RegisterHashFunc(SHA3_224, sumSHA3_224)
	RegisterHashFunc(SHA3_256, sumSHA3_256)
	RegisterHashFunc(SHA3_384, sumSHA3_384)
	RegisterHashFunc(SHA3_512, sumSHA3_512)

	RegisterHashFunc(MURMUR3, sumMURMUR3)

	RegisterHashFunc(SHAKE_128, sumSHAKE128)
	RegisterHashFunc(SHAKE_256, sumSHAKE256)

	// Blake family of hash functions
	// BLAKE2S
	for c := uint64(BLAKE2S_MIN); c <= BLAKE2S_MAX; c++ {
		size := int(c - BLAKE2S_MIN + 1)
		RegisterHashFunc(c, func(buf []byte, _ int) ([]byte, error) {
			return sumBlake2s(buf, size)
		})
	}
	// BLAKE2B
	for c := uint64(BLAKE2B_MIN); c <= BLAKE2B_MAX; c++ {
		size := int(c - BLAKE2B_MIN + 1)
		RegisterHashFunc(c, func(buf []byte, _ int) ([]byte, error) {
			return sumBlake2b(buf, size)
		})
	}
}

func init() {
	registerStdlibHashFuncs()
	registerNonStdlibHashFuncs()
}

// RegisterHashFunc adds an entry to the package-level code -> hash func map.
// The hash function must return at least the requested number of bytes. If it
// returns more, the hash will be truncated.
func RegisterHashFunc(code uint64, hashFunc HashFunc) error {
	if !ValidCode(code) {
		return fmt.Errorf("code %v not valid", code)
	}

	_, ok := funcTable[code]
	if ok {
		return fmt.Errorf("hash func for code %v already registered", code)
	}

	funcTable[code] = hashFunc
	return nil
}
