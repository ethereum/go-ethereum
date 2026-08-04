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

// ToExtWitness converts our internal witness representation to the consensus one.
func (w *Witness) ToExtWitness() *ExtWitness {
	ext := &ExtWitness{
		Headers: slices.Clone(w.Headers),
	}
	slices.Reverse(ext.Headers)

	ext.Codes = make([]hexutil.Bytes, 0, len(w.Codes))
	for code := range w.Codes {
		ext.Codes = append(ext.Codes, []byte(code))
	}
	slices.SortFunc(ext.Codes, func(a, b hexutil.Bytes) int {
		return bytes.Compare(a, b)
	})

	// A binary-tree witness carries its paths, so State and Keys travel as
	// parallel arrays ordered by path. A merkle witness leaves Keys empty and
	// sorts State by content, which is what it is keyed by; that keeps its
	// encoding byte-for-byte what it was before Keys carried anything.
	if len(w.Nodes) > 0 {
		paths := make([]string, 0, len(w.Nodes))
		for path := range w.Nodes {
			paths = append(paths, path)
		}
		slices.Sort(paths)

		ext.Keys = make([]hexutil.Bytes, 0, len(paths))
		ext.State = make([]hexutil.Bytes, 0, len(paths))
		for _, path := range paths {
			ext.Keys = append(ext.Keys, []byte(path))
			ext.State = append(ext.State, w.Nodes[path])
		}
		return ext
	}
	ext.State = make([]hexutil.Bytes, 0, len(w.State))
	for node := range w.State {
		ext.State = append(ext.State, []byte(node))
	}
	slices.SortFunc(ext.State, func(a, b hexutil.Bytes) int {
		return bytes.Compare(a, b)
	})
	return ext
}

// FromExtWitness converts the consensus witness format into our internal one.
func (w *Witness) FromExtWitness(ext *ExtWitness) error {
	if len(ext.Headers) == 0 {
		return errors.New("witness must contain at least one header")
	}
	w.Headers = slices.Clone(ext.Headers)
	// don't trust the input and sort headers in reverse order
	// this is only useful for calling `Root`
	slices.SortFunc(w.Headers, func(a, b *types.Header) int {
		return b.Number.Cmp(a.Number)
	})

	w.Codes = make(map[string]struct{}, len(ext.Codes))
	for _, code := range ext.Codes {
		w.Codes[string(code)] = struct{}{}
	}
	// Keys present means a binary-tree witness: State is path-addressed and
	// the two arrays must line up exactly, so a mismatch is rejected rather
	// than silently truncated to the shorter one.
	if len(ext.Keys) > 0 {
		if len(ext.Keys) != len(ext.State) {
			return fmt.Errorf("witness has %d keys for %d nodes", len(ext.Keys), len(ext.State))
		}
		w.Nodes = make(map[string][]byte, len(ext.Keys))
		for i, path := range ext.Keys {
			if _, dup := w.Nodes[string(path)]; dup {
				return fmt.Errorf("witness repeats the node at path %x", []byte(path))
			}
			w.Nodes[string(path)] = ext.State[i]
		}
		w.State = make(map[string]struct{})
		return nil
	}
	w.State = make(map[string]struct{}, len(ext.State))
	for _, node := range ext.State {
		w.State[string(node)] = struct{}{}
	}
	w.Nodes = make(map[string][]byte)
	return nil
}

// EncodeRLP serializes a witness as RLP.
func (w *Witness) EncodeRLP(wr io.Writer) error {
	return rlp.Encode(wr, w.ToExtWitness())
}

// DecodeRLP decodes a witness from RLP.
func (w *Witness) DecodeRLP(s *rlp.Stream) error {
	var ext ExtWitness
	if err := s.Decode(&ext); err != nil {
		return err
	}
	return w.FromExtWitness(&ext)
}

// ExtWitness is a witness RLP encoding for transferring across clients.
type ExtWitness struct {
	Headers []*types.Header `json:"headers"`
	Codes   []hexutil.Bytes `json:"codes"`
	State   []hexutil.Bytes `json:"state"`
	Keys    []hexutil.Bytes `json:"keys"`
}
