// Copyright 2026 The go-ethereum Authors
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

//go:build zisk

package kzg4844

import (
	"errors"
	"sync"

	"github.com/0xPolygonHermez/zisk/ziskasm/lang/go"
)

var errNotOnZisk = errors.New("kzg4844: only proof verification is available on ZisK")

// gokzgIniter is kept for the dispatcher; there is nothing to initialize.
var gokzgIniter sync.Once

func gokzgInit() {}

func gokzgVerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	ok, err := zkvm.KzgPointEval((*[48]byte)(&commitment), (*[32]byte)(&point), (*[32]byte)(&claim), (*[48]byte)(&proof))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("kzg4844: invalid proof")
	}
	return nil
}

func gokzgBlobToCommitment(blob *Blob) (Commitment, error) { return Commitment{}, errNotOnZisk }

func gokzgComputeProof(blob *Blob, point Point) (Proof, Claim, error) {
	return Proof{}, Claim{}, errNotOnZisk
}

func gokzgComputeBlobProof(blob *Blob, commitment Commitment) (Proof, error) {
	return Proof{}, errNotOnZisk
}

func gokzgVerifyBlobProof(blob *Blob, commitment Commitment, proof Proof) error {
	return errNotOnZisk
}

func gokzgComputeCellProofs(blob *Blob) ([]Proof, error) { return nil, errNotOnZisk }

func gokzgVerifyCellProofBatch(blobs []Blob, commitments []Commitment, cellProofs []Proof) error {
	return errNotOnZisk
}

func gokzgVerifyCells(cells []Cell, commitments []Commitment, cellProofs []Proof, cellIndices []uint64) error {
	return errNotOnZisk
}

func gokzgComputeCells(blobs []Blob) ([]Cell, error) { return nil, errNotOnZisk }

func gokzgRecoverBlobs(cells []Cell, cellIndices []uint64) ([]Blob, error) { return nil, errNotOnZisk }

func gokzgRecoverCells(cells []Cell, cellIndices []uint64) ([]Cell, error) { return nil, errNotOnZisk }
