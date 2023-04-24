package kzg

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"sync"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
)

const (
	BlobCommitmentVersionKZG uint8 = 0x01
	FieldElementsPerBlob     int   = 4096
)

type VersionedHash [32]byte
type Root [32]byte
type Slot uint64

type BlobsSidecar struct {
	BeaconBlockRoot Root
	BeaconBlockSlot Slot
	Blobs           []gokzg4844.Blob
	Proofs          []gokzg4844.KZGProof
}

const (
	PrecompileInputLength = 192
)

var (
	errInvalidInputLength = errors.New("invalid input length")
)

// The value that gets returned when the `verify_kzg_proof“ precompile is called
var precompileReturnValue [64]byte

var gCryptoCtx gokzg4844.Context
var initCryptoCtx sync.Once

// InitializeCryptoCtx initializes the global context object returned via CryptoCtx
func InitializeCryptoCtx() {
	initCryptoCtx.Do(func() {
		// Initialize context to match the configurations that the
		// specs are using.
		ctx, err := gokzg4844.NewContext4096Insecure1337()
		if err != nil {
			panic(fmt.Sprintf("could not create context, err : %v", err))
		}
		gCryptoCtx = *ctx
		// Initialize the precompile return value
		new(big.Int).SetUint64(gokzg4844.ScalarsPerBlob).FillBytes(precompileReturnValue[:32])
		copy(precompileReturnValue[32:], gokzg4844.BlsModulus[:])
	})
}

// CryptoCtx returns a context object stores all of the necessary configurations
// to allow one to create and verify blob proofs.
// This function is expensive to run if the crypto context isn't initialized, so it is recommended to
// pre-initialize by calling InitializeCryptoCtx
func CrpytoCtx() gokzg4844.Context {
	InitializeCryptoCtx()
	return gCryptoCtx
}

// PointEvaluationPrecompile implements point_evaluation_precompile from EIP-4844
func PointEvaluationPrecompile(input []byte) ([]byte, error) {
	if len(input) != PrecompileInputLength {
		return nil, errInvalidInputLength
	}
	// versioned hash: first 32 bytes
	var versionedHash [32]byte
	copy(versionedHash[:], input[:32])

	var x, y gokzg4844.Scalar
	// Evaluation point: next 32 bytes
	copy(x[:], input[32:64])
	// Expected output: next 32 bytes
	copy(y[:], input[64:96])

	// input kzg point: next 48 bytes
	var dataKZG [48]byte
	copy(dataKZG[:], input[96:144])
	if KZGToVersionedHash(gokzg4844.KZGCommitment(dataKZG)) != VersionedHash(versionedHash) {
		return nil, errors.New("mismatched versioned hash")
	}

	// Quotient kzg: next 48 bytes
	var quotientKZG gokzg4844.KZGProof
	copy(quotientKZG[:], input[144:PrecompileInputLength])

	cryptoCtx := CrpytoCtx()
	err := cryptoCtx.VerifyKZGProof(dataKZG, x, y, quotientKZG)
	if err != nil {
		return nil, fmt.Errorf("verify_kzg_proof error: %v", err)
	}

	result := precompileReturnValue // copy the value

	return result[:], nil
}

// KZGToVersionedHash implements kzg_to_versioned_hash from EIP-4844
func KZGToVersionedHash(kzg gokzg4844.KZGCommitment) VersionedHash {
	h := sha256.Sum256(kzg[:])
	h[0] = BlobCommitmentVersionKZG

	return VersionedHash(h)
}
