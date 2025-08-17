// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

const (
	// BlobSidecarVersion0 includes a single proof for verifying the entire blob
	// against its commitment. Used when the full blob is available and needs to
	// be checked as a whole.
	BlobSidecarVersion0 = byte(0)

	// BlobSidecarVersion1 includes multiple cell proofs for verifying specific
	// blob elements (cells). Used in scenarios like data availability sampling,
	// where only portions of the blob are verified individually.
	BlobSidecarVersion1 = byte(1)
)

// BlobTx represents an EIP-4844 transaction.
type BlobTx struct {
	ChainID    *uint256.Int
	Nonce      uint64
	GasTipCap  *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *uint256.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         common.Address
	Value      *uint256.Int
	Data       []byte
	AccessList AccessList
	BlobFeeCap *uint256.Int // a.k.a. maxFeePerBlobGas
	BlobHashes []common.Hash

	// A blob transaction can optionally contain blobs. This field must be set when BlobTx
	// is used to create a transaction for signing.
	Sidecar *BlobTxSidecar `rlp:"-"`

	// Signature values
	V *uint256.Int
	R *uint256.Int
	S *uint256.Int
}

// BlobTxSidecar contains the blobs of a blob transaction.
type BlobTxSidecar struct {
	Version     byte                 // Version
	Blobs       []kzg4844.Blob       // Blobs needed by the blob pool
	Commitments []kzg4844.Commitment // Commitments needed by the blob pool
	Proofs      []kzg4844.Proof      // Proofs needed by the blob pool
}

// NewBlobTxSidecar initialises the BlobTxSidecar object with the provided parameters.
func NewBlobTxSidecar(version byte, blobs []kzg4844.Blob, commitments []kzg4844.Commitment, proofs []kzg4844.Proof) *BlobTxSidecar {
	return &BlobTxSidecar{
		Version:     version,
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}
}

// BlobHashes computes the blob hashes of the given blobs.
func (sc *BlobTxSidecar) BlobHashes() []common.Hash {
	hasher := sha256.New()
	h := make([]common.Hash, len(sc.Commitments))
	for i := range sc.Blobs {
		h[i] = kzg4844.CalcBlobHashV1(hasher, &sc.Commitments[i])
	}
	return h
}

// CellProofsAt returns the cell proofs for blob with index idx.
// This method is only valid for sidecars with version 1.
func (sc *BlobTxSidecar) CellProofsAt(idx int) ([]kzg4844.Proof, error) {
	if sc.Version != BlobSidecarVersion1 {
		return nil, fmt.Errorf("cell proof unsupported, version: %d", sc.Version)
	}
	if idx < 0 || idx >= len(sc.Blobs) {
		return nil, fmt.Errorf("cell proof out of bounds, index: %d, blobs: %d", idx, len(sc.Blobs))
	}
	index := idx * kzg4844.CellProofsPerBlob
	if len(sc.Proofs) < index+kzg4844.CellProofsPerBlob {
		return nil, fmt.Errorf("cell proof is corrupted, index: %d, proofs: %d", idx, len(sc.Proofs))
	}
	return sc.Proofs[index : index+kzg4844.CellProofsPerBlob], nil
}

// ToV1 converts the BlobSidecar to version 1, attaching the cell proofs.
func (sc *BlobTxSidecar) ToV1() error {
	if sc.Version != BlobSidecarVersion1 {
		proofs := make([]kzg4844.Proof, 0, len(sc.Blobs)*kzg4844.CellProofsPerBlob)
		for _, blob := range sc.Blobs {
			cellProofs, err := kzg4844.ComputeCellProofs(&blob)
			if err != nil {
				return err
			}
			proofs = append(proofs, cellProofs...)
		}
		sc.Version = BlobSidecarVersion1
		sc.Proofs = proofs
	}
	return nil
}

// encodedSize computes the RLP size of the sidecar elements. This does NOT return the
// encoded size of the BlobTxSidecar, it's just a helper for tx.Size().
func (sc *BlobTxSidecar) encodedSize() uint64 {
	var blobs, commitments, proofs uint64
	for i := range sc.Blobs {
		blobs += rlp.BytesSize(sc.Blobs[i][:])
	}
	for i := range sc.Commitments {
		commitments += rlp.BytesSize(sc.Commitments[i][:])
	}
	for i := range sc.Proofs {
		proofs += rlp.BytesSize(sc.Proofs[i][:])
	}
	return rlp.ListSize(blobs) + rlp.ListSize(commitments) + rlp.ListSize(proofs)
}

// ValidateBlobCommitmentHashes checks whether the given hashes correspond to the
// commitments in the sidecar
func (sc *BlobTxSidecar) ValidateBlobCommitmentHashes(hashes []common.Hash) error {
	if len(sc.Commitments) != len(hashes) {
		return fmt.Errorf("invalid number of %d blob commitments compared to %d blob hashes", len(sc.Commitments), len(hashes))
	}
	hasher := sha256.New()
	for i, vhash := range hashes {
		computed := kzg4844.CalcBlobHashV1(hasher, &sc.Commitments[i])
		if vhash != computed {
			return fmt.Errorf("blob %d: computed hash %#x mismatches transaction one %#x", i, computed, vhash)
		}
	}
	return nil
}

// Copy returns a deep-copied BlobTxSidecar object.
func (sc *BlobTxSidecar) Copy() *BlobTxSidecar {
	return &BlobTxSidecar{
		Version: sc.Version,

		// The element of these slice is fix-size byte array,
		// therefore slices.Clone will actually deep copy by value.
		Blobs:       slices.Clone(sc.Blobs),
		Commitments: slices.Clone(sc.Commitments),
		Proofs:      slices.Clone(sc.Proofs),
	}
}

// blobTxWithBlobs represents blob tx with its corresponding sidecar.
// This is an interface because sidecars are versioned.
type blobTxWithBlobs interface {
	tx() *BlobTx
	assign(*BlobTxSidecar) error
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *BlobTx) copy() TxData {
	cpy := &BlobTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		BlobHashes: make([]common.Hash, len(tx.BlobHashes)),
		Value:      new(uint256.Int),
		ChainID:    new(uint256.Int),
		GasTipCap:  new(uint256.Int),
		GasFeeCap:  new(uint256.Int),
		BlobFeeCap: new(uint256.Int),
		V:          new(uint256.Int),
		R:          new(uint256.Int),
		S:          new(uint256.Int),
	}
	copy(cpy.AccessList, tx.AccessList)
	copy(cpy.BlobHashes, tx.BlobHashes)

	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.BlobFeeCap != nil {
		cpy.BlobFeeCap.Set(tx.BlobFeeCap)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	if tx.Sidecar != nil {
		cpy.Sidecar = tx.Sidecar.Copy()
	}
	return cpy
}

// accessors for innerTx.
func (tx *BlobTx) txType() byte           { return BlobTxType }
func (tx *BlobTx) chainID() *big.Int      { return tx.ChainID.ToBig() }
func (tx *BlobTx) accessList() AccessList { return tx.AccessList }
func (tx *BlobTx) data() []byte           { return tx.Data }
func (tx *BlobTx) gas() uint64            { return tx.Gas }
func (tx *BlobTx) gasFeeCap() *big.Int    { return tx.GasFeeCap.ToBig() }
func (tx *BlobTx) gasTipCap() *big.Int    { return tx.GasTipCap.ToBig() }
func (tx *BlobTx) gasPrice() *big.Int     { return tx.GasFeeCap.ToBig() }
func (tx *BlobTx) value() *big.Int        { return tx.Value.ToBig() }
func (tx *BlobTx) nonce() uint64          { return tx.Nonce }
func (tx *BlobTx) to() *common.Address    { tmp := tx.To; return &tmp }
func (tx *BlobTx) blobGas() uint64        { return params.BlobTxBlobGasPerBlob * uint64(len(tx.BlobHashes)) }

func (tx *BlobTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap.ToBig())
	}
	tip := dst.Sub(tx.GasFeeCap.ToBig(), baseFee)
	if tip.Cmp(tx.GasTipCap.ToBig()) > 0 {
		tip.Set(tx.GasTipCap.ToBig())
	}
	return tip.Add(tip, baseFee)
}

func (tx *BlobTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V.ToBig(), tx.R.ToBig(), tx.S.ToBig()
}

func (tx *BlobTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID.SetFromBig(chainID)
	tx.V.SetFromBig(v)
	tx.R.SetFromBig(r)
	tx.S.SetFromBig(s)
}

func (tx *BlobTx) withoutSidecar() *BlobTx {
	cpy := *tx
	cpy.Sidecar = nil
	return &cpy
}

func (tx *BlobTx) withSidecar(sideCar *BlobTxSidecar) *BlobTx {
	cpy := *tx
	cpy.Sidecar = sideCar
	return &cpy
}

func (tx *BlobTx) encode(b *bytes.Buffer) error {
	err := rlp.Encode(b, tx)
	if err != nil {
		return err
	}

	type sidecarNoVersion struct {
		Blobs       []kzg4844.Blob       // Blobs needed by the blob pool
		Commitments []kzg4844.Commitment // Commitments needed by the blob pool
		Proofs      []kzg4844.Proof      // Proofs needed by the blob pool
	}

	if sidecar := tx.Sidecar; sidecar != nil {
		if sidecar.Version != BlobSidecarVersion0 {
			err = rlp.Encode(b, sidecar.Version)
		}
		if err == nil {
			err = rlp.Encode(b, &sidecarNoVersion{
				Blobs:       sidecar.Blobs,
				Commitments: sidecar.Commitments,
				Proofs:      sidecar.Proofs,
			})
		}
	}
	return err
}

func (tx *BlobTx) decode(input []byte) error {
	bLen, err := rlp.DecodeBytesV1(input, tx)
	if err != nil {
		return err
	}

	type sidecarNoVersion struct {
		Blobs       []kzg4844.Blob       // Blobs needed by the blob pool
		Commitments []kzg4844.Commitment // Commitments needed by the blob pool
		Proofs      []kzg4844.Proof      // Proofs needed by the blob pool
	}

	// If the input is shorter than the RLP-encoded transaction, it means
	// that the transaction does not have a sidecar.
	if input = input[bLen:]; len(input) > 0 {
		var (
			version = byte(0)
			sidecar = sidecarNoVersion{}
		)
		kind, content, _, err := rlp.Split(input)
		// If the first element is a byte, it means that the sidecar version is encoded.
		if kind == rlp.Byte {
			if err = rlp.DecodeBytes(content, &version); err != nil {
				return err
			}
			input = input[len(content):]
		}
		if err = rlp.DecodeBytes(input, &sidecar); err != nil {
			return err
		}
		tx.Sidecar = &BlobTxSidecar{
			Version:     version,
			Blobs:       sidecar.Blobs,
			Commitments: sidecar.Commitments,
			Proofs:      sidecar.Proofs,
		}
	}
	return nil
}

func (tx *BlobTx) sigHash(chainID *big.Int) common.Hash {
	return prefixedRlpHash(
		BlobTxType,
		[]any{
			chainID,
			tx.Nonce,
			tx.GasTipCap,
			tx.GasFeeCap,
			tx.Gas,
			tx.To,
			tx.Value,
			tx.Data,
			tx.AccessList,
			tx.BlobFeeCap,
			tx.BlobHashes,
		})
}
