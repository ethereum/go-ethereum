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

package kzg4844

import (
	"encoding/json"
	"sync"

	gokzg4844 "github.com/crate-crypto/go-eth-kzg"
)

// context is the crypto primitive pre-seeded with the trusted setup parameters.
var context *gokzg4844.Context

// gokzgIniter ensures that we initialize the KZG library once before using it.
var gokzgIniter sync.Once

// gokzgInit initializes the KZG library with the provided trusted setup.
func gokzgInit() {
	config, err := content.ReadFile("trusted_setup.json")
	if err != nil {
		panic(err)
	}
	params := new(gokzg4844.JSONTrustedSetup)
	if err = json.Unmarshal(config, params); err != nil {
		panic(err)
	}
	context, err = gokzg4844.NewContext4096(params)
	if err != nil {
		panic(err)
	}
}

// gokzgBlobToCommitment creates a small commitment out of a data blob.
func gokzgBlobToCommitment(blob *Blob) (Commitment, error) {
	gokzgIniter.Do(gokzgInit)

	commitment, err := context.BlobToKZGCommitment((*gokzg4844.Blob)(blob), 0)
	if err != nil {
		return Commitment{}, err
	}
	return (Commitment)(commitment), nil
}

// gokzgComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func gokzgComputeProof(blob *Blob, point Point) (Proof, Claim, error) {
	gokzgIniter.Do(gokzgInit)

	proof, claim, err := context.ComputeKZGProof((*gokzg4844.Blob)(blob), (gokzg4844.Scalar)(point), 0)
	if err != nil {
		return Proof{}, Claim{}, err
	}
	return (Proof)(proof), (Claim)(claim), nil
}

// gokzgVerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
func gokzgVerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	gokzgIniter.Do(gokzgInit)

	return context.VerifyKZGProof((gokzg4844.KZGCommitment)(commitment), (gokzg4844.Scalar)(point), (gokzg4844.Scalar)(claim), (gokzg4844.KZGProof)(proof))
}

// gokzgComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func gokzgComputeBlobProof(blob *Blob, commitment Commitment) (Proof, error) {
	gokzgIniter.Do(gokzgInit)

	proof, err := context.ComputeBlobKZGProof((*gokzg4844.Blob)(blob), (gokzg4844.KZGCommitment)(commitment), 0)
	if err != nil {
		return Proof{}, err
	}
	return (Proof)(proof), nil
}

// gokzgVerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func gokzgVerifyBlobProof(blob *Blob, commitment Commitment, proof Proof) error {
	gokzgIniter.Do(gokzgInit)

	return context.VerifyBlobKZGProof((*gokzg4844.Blob)(blob), (gokzg4844.KZGCommitment)(commitment), (gokzg4844.KZGProof)(proof))
}

// gokzgVerifyCellKZGProofBatch verifies a batch of KZG proofs for a set of cells.
func gokzgVerifyCellKZGProofBatch(commitments []Commitment, cellIndicies []uint64, cells []Cell, proofs []Proof) error {
	gokzgIniter.Do(gokzgInit)

	comts := make([]gokzg4844.KZGCommitment, len(commitments))
	for i, commitment := range commitments {
		comts[i] = (gokzg4844.KZGCommitment)(commitment)
	}

	cellInputs := make([]*gokzg4844.Cell, len(cells))
	for i := range cells {
		cellInputs[i] = (*gokzg4844.Cell)(&cells[i])
	}

	proofsInput := make([]gokzg4844.KZGProof, len(proofs))
	for i, proof := range proofs {
		proofsInput[i] = (gokzg4844.KZGProof)(proof)
	}

	return context.VerifyCellKZGProofBatch(comts, cellIndicies, cellInputs, proofsInput)
}

// gokzgComputeCells computes the cells for a given blob.
func gokzgComputeCells(blob *Blob) ([]Cell, error) {
	gokzgIniter.Do(gokzgInit)

	results, err := context.ComputeCells((*gokzg4844.Blob)(blob), 0)
	if err != nil {
		return nil, err
	}

	cells := make([]Cell, len(results))
	for i, result := range results {
		cells[i] = (Cell)(*result)
	}

	return cells, nil
}

// gokzgComputeCellsAndKZGProofs computes the cells and KZG proofs for a given blob.
func gokzgComputeCellsAndKZGProofs(blob *Blob) ([]Cell, []Proof, error) {
	gokzgIniter.Do(gokzgInit)

	cellResults, proofResults, err := context.ComputeCellsAndKZGProofs((*gokzg4844.Blob)(blob), 0)
	if err != nil {
		return nil, nil, err
	}

	cells := make([]Cell, len(cellResults))
	for i, cell := range cellResults {
		cells[i] = (Cell)(*cell)
	}

	proofs := make([]Proof, len(proofResults))
	for i, proof := range proofResults {
		proofs[i] = (Proof)(proof)
	}

	return cells, proofs, nil
}
