// Copyright 2024 The go-ethereum Authors
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

package stateless

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type extWitness -field-override extWitnessMarshalling -out gen_encoding_json.go

// toExtWitness converts our internal witness representation to the consensus one.
func (w *Witness) toExtWitness() *extWitness {
	ext := &extWitness{
		Block:   w.Block,
		Headers: w.Headers,
	}
	ext.Codes = make([][]byte, 0, len(w.Codes))
	for code := range w.Codes {
		ext.Codes = append(ext.Codes, []byte(code))
	}
	slices.SortFunc(ext.Codes, bytes.Compare)

	ext.State = make([][]byte, 0, len(w.State))
	for node := range w.State {
		ext.State = append(ext.State, []byte(node))
	}
	slices.SortFunc(ext.State, bytes.Compare)
	return ext
}

// fromExtWitness converts the consensus witness format into our internal one.
func (w *Witness) fromExtWitness(ext *extWitness) error {
	w.Block, w.Headers = ext.Block, ext.Headers

	w.Codes = make(map[string]struct{}, len(ext.Codes))
	for _, code := range ext.Codes {
		w.Codes[string(code)] = struct{}{}
	}
	w.State = make(map[string]struct{}, len(ext.State))
	for _, node := range ext.State {
		w.State[string(node)] = struct{}{}
	}
	return w.sanitize()
}

// MarshalJSON marshals a witness as JSON.
func (w *Witness) MarshalJSON() ([]byte, error) {
	return w.toExtWitness().MarshalJSON()
}

// EncodeRLP serializes a witness as RLP.
func (w *Witness) EncodeRLP(wr io.Writer) error {
	return rlp.Encode(wr, w.toExtWitness())
}

// UnmarshalJSON unmarshals from JSON.
func (w *Witness) UnmarshalJSON(input []byte) error {
	var ext extWitness
	if err := ext.UnmarshalJSON(input); err != nil {
		return err
	}
	return w.fromExtWitness(&ext)
}

// DecodeRLP decodes a witness from RLP.
func (w *Witness) DecodeRLP(s *rlp.Stream) error {
	var ext extWitness
	if err := s.Decode(&ext); err != nil {
		return err
	}
	return w.fromExtWitness(&ext)
}

// sanitize checks for some mandatory fields in the witness after decoding so
// the rest of the code can assume invariants and doesn't have to deal with
// corrupted data.
func (w *Witness) sanitize() error {
	// Verify that the "parent" header (i.e. index 0) is available, and is the
	// true parent of the block-to-be executed, since we use that to link the
	// current block to the pre-state.
	if len(w.Headers) == 0 {
		return errors.New("parent header (for pre-root hash) missing")
	}
	for i, header := range w.Headers {
		if header == nil {
			return fmt.Errorf("witness header nil at position %d", i)
		}
	}
	if w.Headers[0].Hash() != w.Block.ParentHash() {
		return fmt.Errorf("parent hash different: witness %v, block parent %v", w.Headers[0].Hash(), w.Block.ParentHash())
	}
	return nil
}

// extWitness is a witness RLP encoding for transferring across clients.
type extWitness struct {
	Block   *types.Block    `json:"block"       gencodec:"required"`
	Headers []*types.Header `json:"headers"       gencodec:"required"`
	Codes   [][]byte        `json:"codes"`
	State   [][]byte        `json:"state"`
}

// extWitnessMarshalling defines the hex marshalling types for a witness.
type extWitnessMarshalling struct {
	Codes []hexutil.Bytes
	State []hexutil.Bytes
}
