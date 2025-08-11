// Copyright 2025 The go-ethereum Authors
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
package era

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type variant uint16

const (
	proofNone variant = iota
	proofHistoricalHashesAccumulator
	proofHistoricalRoots
	proofCapella
	proofDeneb
)

type BlockProofHistoricalHashesAccumulator [15]common.Hash // 15 * 32 = 480 bytes

// BlockProofHistoricalRoots – Altair / Bellatrix historical_roots branch.
type BlockProofHistoricalRoots struct {
	BeaconBlockProof    [14]common.Hash // 448
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [11]common.Hash // 352
	Slot                uint64          // 8  => 840 bytes
}

// BlockProofHistoricalSummariesCapella – Capella historical_summaries branch.
type BlockProofHistoricalSummariesCapella struct {
	BeaconBlockProof    [13]common.Hash // 416
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [11]common.Hash // 352
	Slot                uint64          // 8  => 808 bytes
}

// BlockProofHistoricalSummariesDeneb – Deneb historical_summaries branch.
type BlockProofHistoricalSummariesDeneb struct {
	BeaconBlockProof    [13]common.Hash // 416
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [12]common.Hash // 384
	Slot                uint64          // 8  => 840 bytes
}

// Proof is the interface for all block proof types in the era2 package.
type Proof interface {
	EncodeRLP(w io.Writer) error
	DecodeRlP(s *rlp.Stream) error
	Variant() variant
}

type hhaAlias BlockProofHistoricalHashesAccumulator

// EncodeRLP encodes the BlockProofHistoricalHashesAccumulator into RLP format.
func (p *BlockProofHistoricalHashesAccumulator) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofHistoricalHashesAccumulator), hhaAlias(*p)}
	return rlp.Encode(w, payload)
}

// Variant returns the variant type of the BlockProofHistoricalHashesAccumulator.
func (p *BlockProofHistoricalHashesAccumulator) Variant() variant {
	return proofHistoricalHashesAccumulator
}

type rootsAlias BlockProofHistoricalRoots

// EncodeRLP encodes the BlockProofHistoricalRoots into RLP format.
func (p *BlockProofHistoricalRoots) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofHistoricalRoots), rootsAlias(*p)}
	return rlp.Encode(w, payload)
}

// Variant returns the variant type of the BlockProofHistoricalRoots.
func (*BlockProofHistoricalRoots) Variant() variant { return proofHistoricalRoots }

type capellaAlias BlockProofHistoricalSummariesCapella

// EncodeRLP encodes the BlockProofHistoricalSummariesCapella into RLP format.
func (p *BlockProofHistoricalSummariesCapella) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofCapella), capellaAlias(*p)}
	return rlp.Encode(w, payload)
}

// Variant returns the variant type of the BlockProofHistoricalSummariesCapella.
func (*BlockProofHistoricalSummariesCapella) Variant() variant { return proofCapella }

type denebAlias BlockProofHistoricalSummariesDeneb

// EncodeRLP encodes the BlockProofHistoricalSummariesDeneb into RLP format.
func (p *BlockProofHistoricalSummariesDeneb) EncodeRLP(w io.Writer) error {
	payload := []interface{}{uint16(proofDeneb), denebAlias(*p)}
	return rlp.Encode(w, payload)
}

// Variant returns the variant type of the BlockProofHistoricalSummariesDeneb.
func (*BlockProofHistoricalSummariesDeneb) Variant() variant { return proofDeneb }
