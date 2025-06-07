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
	"errors"
	"fmt"
	"math/big"

	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
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
func (sc *BlobTxSidecar) CellProofsAt(idx int) []kzg4844.Proof {
	var cellProofs []kzg4844.Proof
	for i := range kzg4844.CellProofsPerBlob {
		index := idx*kzg4844.CellProofsPerBlob + i
		if index > len(sc.Proofs) {
			return nil
		}
		proof := sc.Proofs[index]
		cellProofs = append(cellProofs, proof)
	}
	return cellProofs
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

// blobTxWithBlobs represents blob tx with its corresponding sidecar.
// This is an interface because sidecars are versioned.
type blobTxWithBlobs interface {
	tx() *BlobTx
	assign(*BlobTxSidecar) error
}

type blobTxWithBlobsV0 struct {
	BlobTx      *BlobTx
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

type blobTxWithBlobsV1 struct {
	BlobTx      *BlobTx
	Version     byte
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

func (btx *blobTxWithBlobsV0) tx() *BlobTx {
	return btx.BlobTx
}

func (btx *blobTxWithBlobsV0) assign(sc *BlobTxSidecar) error {
	sc.Version = 0
	sc.Blobs = btx.Blobs
	sc.Commitments = btx.Commitments
	sc.Proofs = btx.Proofs
	return nil
}

func (btx *blobTxWithBlobsV1) tx() *BlobTx {
	return btx.BlobTx
}

func (btx *blobTxWithBlobsV1) assign(sc *BlobTxSidecar) error {
	if btx.Version != 1 {
		return fmt.Errorf("unsupported blob tx version %d", btx.Version)
	}
	sc.Version = 1
	sc.Blobs = btx.Blobs
	sc.Commitments = btx.Commitments
	sc.Proofs = btx.Proofs
	return nil
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
		cpy.Sidecar = &BlobTxSidecar{
			Blobs:       slices.Clone(tx.Sidecar.Blobs),
			Commitments: slices.Clone(tx.Sidecar.Commitments),
			Proofs:      slices.Clone(tx.Sidecar.Proofs),
		}
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
	switch {
	case tx.Sidecar == nil:
		return rlp.Encode(b, tx)

	case tx.Sidecar.Version == 0:
		return rlp.Encode(b, &blobTxWithBlobsV0{
			BlobTx:      tx,
			Blobs:       tx.Sidecar.Blobs,
			Commitments: tx.Sidecar.Commitments,
			Proofs:      tx.Sidecar.Proofs,
		})

	case tx.Sidecar.Version == 1:
		return rlp.Encode(b, &blobTxWithBlobsV1{
			BlobTx:      tx,
			Version:     tx.Sidecar.Version,
			Blobs:       tx.Sidecar.Blobs,
			Commitments: tx.Sidecar.Commitments,
			Proofs:      tx.Sidecar.Proofs,
		})

	default:
		return errors.New("unsupported sidecar version")
	}
}

func (tx *BlobTx) decode(input []byte) error {
	// Here we need to support two outer formats: the network protocol encoding of the tx
	// (with blobs) or the canonical encoding without blobs.
	//
	// The canonical encoding is just a list of fields:
	//
	//     [chainID, nonce, ...]
	//
	// The network encoding is a list where the first element is the tx in the canonical encoding,
	// and the remaining elements are the 'sidecar':
	//
	//     [[chainID, nonce, ...], ...]
	//
	// The two outer encodings can be distinguished by checking whether the first element
	// of the input list is itself a list. If it's the canonical encoding, the first
	// element is the chainID, which is a number.

	firstElem, _, err := rlp.SplitList(input)
	if err != nil {
		return err
	}
	firstElemKind, _, secondElem, err := rlp.Split(firstElem)
	if err != nil {
		return err
	}
	if firstElemKind != rlp.List {
		// Blob tx without blobs.
		return rlp.DecodeBytes(input, tx)
	}

	// Now we know it's the network encoding with the blob sidecar. Here we again need to
	// support multiple encodings: legacy sidecars (v0) with a blob proof, and versioned
	// sidecars.
	//
	// The legacy encoding is:
	//
	//     [tx, blobs, commitments, proofs]
	//
	// The versioned encoding is:
	//
	//     [tx, version, blobs, ...]
	//
	// We can tell the two apart by checking whether the second element is the version byte.
	// For legacy sidecar the second element is a list of blobs.

	secondElemKind, _, _, err := rlp.Split(secondElem)
	if err != nil {
		return err
	}
	var payload blobTxWithBlobs
	if secondElemKind == rlp.List {
		// No version byte: blob sidecar v0.
		payload = new(blobTxWithBlobsV0)
	} else {
		// It has a version byte. Decode as v1, version is checked by assign()
		payload = new(blobTxWithBlobsV1)
	}
	if err := rlp.DecodeBytes(input, payload); err != nil {
		return err
	}
	sc := new(BlobTxSidecar)
	if err := payload.assign(sc); err != nil {
		return err
	}
	*tx = *payload.tx()
	tx.Sidecar = sc
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
