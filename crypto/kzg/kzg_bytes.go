package kzg

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/params"
	"github.com/protolambda/go-kzg/bls"
	"github.com/protolambda/ztyp/codec"
)

// The custom types from EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/polynomial-commitments.md#custom-types
type KZGCommitment [48]byte
type KZGProof [48]byte
type VersionedHash [32]byte
type Root [32]byte
type Slot uint64

type BlobsSidecar struct {
	BeaconBlockRoot    Root
	BeaconBlockSlot    Slot
	Blobs              BlobSequence
	KZGAggregatedProof KZGProof
}

type BlobSequence interface {
	Len() int
	At(int) Blob
}

type Blob interface {
	Len() int
	At(int) [32]byte
}

type KZGCommitmentSequence interface {
	Len() int
	At(int) KZGCommitment
}

type KZGCommitmentSequenceImpl []KZGCommitment

func (s KZGCommitmentSequenceImpl) At(i int) KZGCommitment {
	return s[i]
}

func (s KZGCommitmentSequenceImpl) Len() int {
	return len(s)
}

const (
	BlobTxType                = 5
	PrecompileInputLength     = 192
	BlobVersionedHashesOffset = 258 // offset the versioned_hash_offset in a serialized blob tx

	blobMessageLen = 192 // size in bytes of "message" within the serialized blob tx
)

var (
	invalidKZGProofError = errors.New("invalid kzg proof")
)

// PointEvaluationPrecompile implements point_evaluation_precompile from EIP-4844
func PointEvaluationPrecompile(input []byte) ([]byte, error) {
	if len(input) != PrecompileInputLength {
		return nil, errors.New("invalid input length")
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
	if KZGToVersionedHash(KZGCommitment(dataKZG)) != VersionedHash(versionedHash) {
		return nil, errors.New("mismatched versioned hash")
	}

	// Quotient kzg: next 48 bytes
	var quotientKZG [48]byte
	copy(quotientKZG[:], input[144:PrecompileInputLength])

	ok, err := VerifyKZGProof(KZGCommitment(dataKZG), x, y, KZGProof(quotientKZG))
	if err != nil {
		return nil, fmt.Errorf("verify_kzg_proof error: %v", err)
	}
	if !ok {
		return nil, invalidKZGProofError
	}
	return []byte{}, nil
}

// VerifyKZGProof implements verify_kzg_proof from the EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/polynomial-commitments.md#verify_kzg_proof
func VerifyKZGProof(polynomialKZG KZGCommitment, z, y [32]byte, kzgProof KZGProof) (bool, error) {
	// successfully converting z and y to bls.Fr confirms they are < MODULUS per the spec
	var zFr, yFr bls.Fr
	ok := bls.FrFrom32(&zFr, z)
	if !ok {
		return false, errors.New("invalid evaluation point")
	}
	ok = bls.FrFrom32(&yFr, y)
	if !ok {
		return false, errors.New("invalid expected output")
	}
	polynomialKZGG1, err := bls.FromCompressedG1(polynomialKZG[:])
	if err != nil {
		return false, fmt.Errorf("failed to decode polynomialKZG: %v", err)
	}
	kzgProofG1, err := bls.FromCompressedG1(kzgProof[:])
	if err != nil {
		return false, fmt.Errorf("failed to decode kzgProof: %v", err)
	}
	return VerifyKZGProofFromPoints(polynomialKZGG1, &zFr, &yFr, kzgProofG1), nil
}

// KZGToVersionedHash implements kzg_to_versioned_hash from EIP-4844
func KZGToVersionedHash(kzg KZGCommitment) VersionedHash {
	h := sha256.Sum256(kzg[:])
	h[0] = params.BlobCommitmentVersionKZG
	return VersionedHash([32]byte(h))
}

// BlobToKZGCommitment implements blob_to_kzg_commitment from the EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/polynomial-commitments.md#blob_to_kzg_commitment
func BlobToKZGCommitment(blob Blob) (KZGCommitment, bool) {
	poly, ok := BlobToPolynomial(blob)
	if !ok {
		return KZGCommitment{}, false
	}
	return PolynomialToKZGCommitment(poly), true
}

// VerifyAggregateKZGProof implements verify_aggregate_kzg_proof from the EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/polynomial-commitments.md#verify_aggregate_kzg_proof
func VerifyAggregateKZGProof(blobs BlobSequence, expectedKZGCommitments KZGCommitmentSequence, kzgAggregatedProof KZGProof) (bool, error) {
	polynomials, ok := BlobsToPolynomials(blobs)
	if !ok {
		return false, errors.New("could not convert blobs to polynomials")
	}
	aggregatedPoly, aggregatedPolyCommitment, evaluationChallenge, err :=
		ComputeAggregatedPolyAndCommitment(polynomials, expectedKZGCommitments)
	if err != nil {
		return false, err
	}
	y := EvaluatePolynomialInEvaluationForm(aggregatedPoly, evaluationChallenge)
	kzgProofG1, err := bls.FromCompressedG1(kzgAggregatedProof[:])
	if err != nil {
		return false, fmt.Errorf("failed to decode kzgProof: %v", err)
	}
	return VerifyKZGProofFromPoints(aggregatedPolyCommitment, evaluationChallenge, y, kzgProofG1), nil
}

// ComputeAggregateKZGProof implements compute_aggregate_kzg_proof from the EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/polynomial-commitments.md#compute_aggregate_kzg_proof
func ComputeAggregateKZGProof(blobs BlobSequence) (KZGProof, error) {
	polynomials, ok := BlobsToPolynomials(blobs)
	if !ok {
		return KZGProof{}, errors.New("could not convert blobs to polynomials")
	}
	return ComputeAggregateKZGProofFromPolynomials(polynomials)
}

// ValidateBlobsSidecar implements validate_blobs_sidecar from the EIP-4844 consensus spec:
// https://github.com/roberto-bayardo/consensus-specs/blob/dev/specs/eip4844/beacon-chain.md#validate_blobs_sidecar
func ValidateBlobsSidecar(slot Slot, beaconBlockRoot Root, expectedKZGCommitments KZGCommitmentSequence, blobsSidecar BlobsSidecar) error {
	if slot != blobsSidecar.BeaconBlockSlot {
		return fmt.Errorf(
			"slot doesn't match sidecar's beacon block slot (%v != %v)",
			slot, blobsSidecar.BeaconBlockSlot)
	}
	if beaconBlockRoot != blobsSidecar.BeaconBlockRoot {
		return errors.New("roots not equal")
	}
	blobs := blobsSidecar.Blobs
	if blobs.Len() != expectedKZGCommitments.Len() {
		return fmt.Errorf(
			"blob len doesn't match expected kzg commitments len (%v != %v)",
			blobs.Len(), expectedKZGCommitments.Len())
	}
	ok, err := VerifyAggregateKZGProof(blobs, expectedKZGCommitments, blobsSidecar.KZGAggregatedProof)
	if err != nil {
		return fmt.Errorf("verify_aggregate_kzg_proof error: %v", err)
	}
	if !ok {
		return invalidKZGProofError
	}
	return nil
}

// TxPeekBlobVersionedHashes implements tx_peek_blob_versioned_hashes from EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/beacon-chain.md#tx_peek_blob_versioned_hashes
//
// Format of the blob tx relevant to this function is as follows:
//   0: type (value should always be BlobTxType, 1 byte)
//   1: message offset (value should always be 69, 4 bytes)
//   5: ECDSA signature (65 bytes)
//   70: start of "message" (192 bytes)
//     258: start of the versioned hash offset within "message"  (4 bytes)
//   262-: rest of the tx following message
func TxPeekBlobVersionedHashes(tx []byte) ([]VersionedHash, error) {
	// we start our reader at the versioned hash offset within the serialized tx
	if len(tx) < BlobVersionedHashesOffset {
		return nil, errors.New("blob tx invalid: too short")
	}
	if tx[0] != BlobTxType {
		return nil, errors.New("invalid blob tx type")
	}
	dr := codec.NewDecodingReader(bytes.NewReader(tx[BlobVersionedHashesOffset:]), uint64(len(tx)-BlobVersionedHashesOffset))

	// read the offset to the versioned hashes
	var offset uint32
	offset, err := dr.ReadOffset()
	if err != nil {
		return nil, fmt.Errorf("could not read versioned hashes offset: %v", err)
	}

	// Advance dr to the versioned hash list. We subtract blobMessageLen from the offset here to
	// account for the fact that the offset is relative to the position of "message" (70) and we
	// are currently positioned at the end of it (262).
	skip := uint64(offset) - blobMessageLen
	skipped, err := dr.Skip(skip)
	if err != nil {
		return nil, fmt.Errorf("could not skip to versioned hashes: %v", err)
	}
	if skip != uint64(skipped) {
		return nil, fmt.Errorf("did not skip to versioned hashes. want %v got %v", skip, skipped)
	}

	// read the list of hashes one by one until we hit the end of the data
	hashes := []VersionedHash{}
	tmp := make([]byte, 32)
	for dr.Scope() > 0 {
		if _, err = dr.Read(tmp); err != nil {
			return nil, fmt.Errorf("could not read versioned hashes: %v", err)
		}
		var h VersionedHash
		copy(h[:], tmp)
		hashes = append(hashes, h)
	}

	return hashes, nil
}

// VerifyKZGCommitmentsAgainstTransactions implements verify_kzg_commitments_against_transactions
// from the EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/beacon-chain.md#verify_kzg_commitments_against_transactions
func VerifyKZGCommitmentsAgainstTransactions(transactions [][]byte, kzgCommitments KZGCommitmentSequence) error {
	var versionedHashes []VersionedHash
	for _, tx := range transactions {
		if tx[0] == BlobTxType {
			v, err := TxPeekBlobVersionedHashes(tx)
			if err != nil {
				return err
			}
			versionedHashes = append(versionedHashes, v...)
		}
	}
	if kzgCommitments.Len() != len(versionedHashes) {
		return fmt.Errorf("invalid number of blob versioned hashes: %v vs %v", kzgCommitments.Len(), len(versionedHashes))
	}
	for i := 0; i < kzgCommitments.Len(); i++ {
		h := KZGToVersionedHash(kzgCommitments.At(i))
		if h != versionedHashes[i] {
			return errors.New("invalid version hashes vs kzg")
		}
	}
	return nil
}
