// Copyright 2017 The go-ethereum Authors
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
package ethash

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// API exposes ethash related methods for the RPC interface
type API struct {
	ethash *Ethash
}

// GetWork returns a work package for external miner. The work package consists of 3 strings
// result[0], 32 bytes hex encoded current block header pow-hash
// result[1], 32 bytes hex encoded seed hash used for DAG
// result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (s *API) GetWork() ([3]string, error) {
	var (
		workCh = make(chan [3]string)
		errCh  = make(chan error)
	)
	op := &remoteOp{
		typ:    getWork,
		workCh: workCh,
		errCh:  errCh,
	}
	s.ethash.remoteOp <- op
	err := <-errCh
	if err == nil {
		return <-workCh, nil
	} else {
		return [3]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution. It returns an indication if the work was
// accepted. Note, this is not an indication if the provided work was valid!
func (s *API) SubmitWork(nonce types.BlockNonce, solution, digest common.Hash) bool {
	var errCh = make(chan error)
	result := sealResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      solution,
	}
	op := &remoteOp{
		typ:    submitWork,
		result: result,
		errCh:  errCh,
	}
	s.ethash.remoteOp <- op
	if err := <-errCh; err == nil {
		return true
	} else {
		return false
	}
}
