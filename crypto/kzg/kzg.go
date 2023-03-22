package kzg

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"github.com/crate-crypto/go-proto-danksharding-crypto/api"
	"github.com/crate-crypto/go-proto-danksharding-crypto/serialization"
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
	Blobs           []serialization.Blob
	Proofs          []serialization.KZGProof
}

const (
	BlobTxType                = 5
	PrecompileInputLength     = 192
	BlobVersionedHashesOffset = 258 // position of blob_versioned_hashes offset in a serialized blob tx, see TxPeekBlobVersionedHashes
)

var (
	errInvalidInputLength = errors.New("invalid input length")
)

// The value that gets returned when the `verify_kzg_proof“ precompile is called
var precompileReturnValue [64]byte

// The context object stores all of the necessary configurations
// to allow one to create and verify blob proofs
var CryptoCtx api.Context

func init() {
	// Initialize context to match the configurations that the
	// specs are using.
	ctx, err := api.NewContext4096Insecure1337()
	if err != nil {
		panic(fmt.Sprintf("could not create context, err : %v", err))
	}
	CryptoCtx = *ctx
	// Initialize the precompile return value
	new(big.Int).SetUint64(serialization.ScalarsPerBlob).FillBytes(precompileReturnValue[:32])
	copy(precompileReturnValue[32:], api.MODULUS[:])
}

// PointEvaluationPrecompile implements point_evaluation_precompile from EIP-4844
func PointEvaluationPrecompile(input []byte) ([]byte, error) {
	if len(input) != PrecompileInputLength {
		return nil, errInvalidInputLength
	}
	// versioned hash: first 32 bytes
	var versionedHash [32]byte
	copy(versionedHash[:], input[:32])

	var x, y [32]byte
	// Evaluation point: next 32 bytes
	copy(x[:], input[32:64])
	// Expected output: next 32 bytes
	copy(y[:], input[64:96])

	// input kzg point: next 48 bytes
	var dataKZG [48]byte
	copy(dataKZG[:], input[96:144])
	if KZGToVersionedHash(serialization.KZGCommitment(dataKZG)) != VersionedHash(versionedHash) {
		return nil, errors.New("mismatched versioned hash")
	}

	// Quotient kzg: next 48 bytes
	var quotientKZG [48]byte
	copy(quotientKZG[:], input[144:PrecompileInputLength])

	err := CryptoCtx.VerifyKZGProof(dataKZG, quotientKZG, x, y)
	if err != nil {
		return nil, fmt.Errorf("verify_kzg_proof error: %v", err)
	}

	result := precompileReturnValue // copy the value

	return result[:], nil
}

// KZGToVersionedHash implements kzg_to_versioned_hash from EIP-4844
func KZGToVersionedHash(kzg serialization.KZGCommitment) VersionedHash {
	h := sha256.Sum256(kzg[:])
	h[0] = BlobCommitmentVersionKZG

	return VersionedHash(h)
}
