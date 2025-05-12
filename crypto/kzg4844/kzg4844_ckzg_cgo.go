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

//go:build ckzg && !nacl && !js && !wasip1 && cgo && !gofuzz

package kzg4844

import (
	"encoding/json"
	"errors"
	"sync"

	gokzg4844 "github.com/crate-crypto/go-eth-kzg"
	ckzg4844 "github.com/ethereum/c-kzg-4844/v2/bindings/go"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ckzgAvailable signals whether the library was compiled into Geth.
const ckzgAvailable = true

// ckzgIniter ensures that we initialize the KZG library once before using it.
var ckzgIniter sync.Once

// ckzgInit initializes the KZG library with the provided trusted setup.
func ckzgInit() {
	config, err := content.ReadFile("trusted_setup.json")
	if err != nil {
		panic(err)
	}
	params := new(gokzg4844.JSONTrustedSetup)
	if err = json.Unmarshal(config, params); err != nil {
		panic(err)
	}
	if err = gokzg4844.CheckTrustedSetupIsWellFormed(params); err != nil {
		panic(err)
	}
	g1Lag := make([]byte, len(params.SetupG1Lagrange)*(len(params.SetupG1Lagrange[0])-2)/2)
	for i, g1 := range params.SetupG1Lagrange {
		copy(g1Lag[i*(len(g1)-2)/2:], hexutil.MustDecode(g1))
	}
	g1s := make([]byte, len(params.SetupG1Monomial)*(len(params.SetupG1Monomial[0])-2)/2)
	for i, g1 := range params.SetupG1Monomial {
		copy(g1s[i*(len(g1)-2)/2:], hexutil.MustDecode(g1))
	}
	g2s := make([]byte, len(params.SetupG2)*(len(params.SetupG2[0])-2)/2)
	for i, g2 := range params.SetupG2 {
		copy(g2s[i*(len(g2)-2)/2:], hexutil.MustDecode(g2))
	}
	// The last parameter determines the multiplication table, see https://notes.ethereum.org/@jtraglia/windowed_multiplications
	// I think 6 is an decent compromise between size and speed
	if err = ckzg4844.LoadTrustedSetup(g1s, g1Lag, g2s, 6); err != nil {
		panic(err)
	}
}

// ckzgBlobToCommitment creates a small commitment out of a data blob.
func ckzgBlobToCommitment(blob *Blob) (Commitment, error) {
	ckzgIniter.Do(ckzgInit)

	commitment, err := ckzg4844.BlobToKZGCommitment((*ckzg4844.Blob)(blob))
	if err != nil {
		return Commitment{}, err
	}
	return (Commitment)(commitment), nil
}

// ckzgComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func ckzgComputeProof(blob *Blob, point Point) (Proof, Claim, error) {
	ckzgIniter.Do(ckzgInit)

	proof, claim, err := ckzg4844.ComputeKZGProof((*ckzg4844.Blob)(blob), (ckzg4844.Bytes32)(point))
	if err != nil {
		return Proof{}, Claim{}, err
	}
	return (Proof)(proof), (Claim)(claim), nil
}

// ckzgVerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
func ckzgVerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	ckzgIniter.Do(ckzgInit)

	valid, err := ckzg4844.VerifyKZGProof((ckzg4844.Bytes48)(commitment), (ckzg4844.Bytes32)(point), (ckzg4844.Bytes32)(claim), (ckzg4844.Bytes48)(proof))
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proof")
	}
	return nil
}

// ckzgComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func ckzgComputeBlobProof(blob *Blob, commitment Commitment) (Proof, error) {
	ckzgIniter.Do(ckzgInit)

	proof, err := ckzg4844.ComputeBlobKZGProof((*ckzg4844.Blob)(blob), (ckzg4844.Bytes48)(commitment))
	if err != nil {
		return Proof{}, err
	}
	return (Proof)(proof), nil
}

// ckzgVerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func ckzgVerifyBlobProof(blob *Blob, commitment Commitment, proof Proof) error {
	ckzgIniter.Do(ckzgInit)

	valid, err := ckzg4844.VerifyBlobKZGProof((*ckzg4844.Blob)(blob), (ckzg4844.Bytes48)(commitment), (ckzg4844.Bytes48)(proof))
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proof")
	}
	return nil
}

// ckzgComputeCellProofs returns the KZG cell proofs that are used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func ckzgComputeCellProofs(blob *Blob) ([]Proof, error) {
	ckzgIniter.Do(ckzgInit)

	_, proofs, err := ckzg4844.ComputeCellsAndKZGProofs((*ckzg4844.Blob)(blob))
	if err != nil {
		return []Proof{}, err
	}
	var p []Proof
	for _, proof := range proofs {
		p = append(p, (Proof)(proof))
	}
	return p, nil
}

// ckzgVerifyCellProofs verifies that the blob data corresponds to the provided commitment.
func ckzgVerifyCellProofBatch(blobs []*Blob, commitments []Commitment, cellProofs []Proof) error {
	ckzgIniter.Do(ckzgInit)
	var (
		proofs      = make([]ckzg4844.Bytes48, len(cellProofs))
		commits     = make([]ckzg4844.Bytes48, 0, len(cellProofs))
		cellIndices = make([]uint64, 0, len(cellProofs))
		cells       = make([]ckzg4844.Cell, 0, len(cellProofs))
	)
	// Copy over the cell proofs
	for i, proof := range cellProofs {
		proofs[i] = (ckzg4844.Bytes48)(proof)
	}
	// Blow up the commitments to be the same length as the proofs
	for _, commitment := range commitments {
		for range gokzg4844.CellsPerExtBlob {
			commits = append(commits, (ckzg4844.Bytes48)(commitment))
		}
	}
	// Compute the cells and cell indices
	for _, blob := range blobs {
		cellsI, err := ckzg4844.ComputeCells((*ckzg4844.Blob)(blob))
		if err != nil {
			return err
		}
		cells = append(cells, cellsI[:]...)
		for idx := range len(cellsI) {
			cellIndices = append(cellIndices, uint64(idx))
		}
	}

	valid, err := ckzg4844.VerifyCellKZGProofBatch(commits, cellIndices, cells, proofs)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proof")
	}
	return nil
}
