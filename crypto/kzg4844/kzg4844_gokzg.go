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

// gokzgComputeCellProofs returns the KZG cell proofs that are used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func gokzgComputeCellProofs(blob *Blob) ([]Proof, error) {
	gokzgIniter.Do(gokzgInit)

	_, proofs, err := context.ComputeCellsAndKZGProofs((*gokzg4844.Blob)(blob), 0)
	if err != nil {
		return []Proof{}, err
	}
	var p []Proof
	for _, proof := range proofs {
		p = append(p, (Proof)(proof))
	}
	return p, nil
}

// gokzgVerifyCellProofs verifies that the blob data corresponds to the provided commitment.
func gokzgVerifyCellProofBatch(blobs []*Blob, commitments []Commitment, cellProofs []Proof) error {
	gokzgIniter.Do(gokzgInit)

	var (
		proofs      = make([]gokzg4844.KZGProof, len(cellProofs))
		commits     = make([]gokzg4844.KZGCommitment, 0, len(cellProofs))
		cellIndices = make([]uint64, 0, len(cellProofs))
		cells       = make([]*gokzg4844.Cell, 0, len(cellProofs))
	)
	// Copy over the cell proofs
	for i, proof := range cellProofs {
		proofs[i] = gokzg4844.KZGProof(proof)
	}
	// Blow up the commitments to be the same length as the proofs
	for _, commitment := range commitments {
		for range gokzg4844.CellsPerExtBlob {
			commits = append(commits, gokzg4844.KZGCommitment(commitment))
		}
	}
	// Compute the cell and cell indices
	for _, blob := range blobs {
		cellsI, err := context.ComputeCells((*gokzg4844.Blob)(blob), 2)
		if err != nil {
			return err
		}
		cells = append(cells, cellsI[:]...)
		for idx := range len(cellsI) {
			cellIndices = append(cellIndices, uint64(idx))
		}
	}
	return context.VerifyCellKZGProofBatch(commits, cellIndices, cells[:], proofs)
}
