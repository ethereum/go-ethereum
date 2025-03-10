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

//xgo:build ckzg && !nacl && !js && !wasip1 && cgo && !gofuzz

package kzg4844

import (
	"encoding/json"
	"errors"
	"sync"

	goethkzg "github.com/crate-crypto/go-eth-kzg"
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
	params := new(goethkzg.JSONTrustedSetup)
	if err = json.Unmarshal(config, params); err != nil {
		panic(err)
	}
	if err = goethkzg.CheckTrustedSetupIsWellFormed(params); err != nil {
		panic(err)
	}
	g1MonomialBytes := make([]byte, len(params.SetupG1Monomial)*(len(params.SetupG1Monomial[0])-2)/2)
	for i, g1 := range &params.SetupG1Monomial {
		copy(g1MonomialBytes[i*(len(g1)-2)/2:], hexutil.MustDecode(g1))
	}
	g1LagrangeBytes := make([]byte, len(params.SetupG1Lagrange)*(len(params.SetupG1Lagrange[0])-2)/2)
	for i, g1 := range params.SetupG1Lagrange {
		copy(g1LagrangeBytes[i*(len(g1)-2)/2:], hexutil.MustDecode(g1))
	}
	g2MonomialBytes := make([]byte, len(params.SetupG2)*(len(params.SetupG2[0])-2)/2)
	for i, g2 := range params.SetupG2 {
		copy(g2MonomialBytes[i*(len(g2)-2)/2:], hexutil.MustDecode(g2))
	}

	// c-kzg-4844 uses a global context/setup file, so it needs to be 8 for both the EL and CL (currently CL sets it at 8)
	//
	// the parameter is mainly for optimization purposes. Heuristically 8 has been shown to be the best tradeoff
	// It essentially stores something like 2^8 * len(g1_monomial) points
	// So the higher the number, the more memory it uses
	// You can set it to 0 to store nothing and it uses a different codepath, but its slow
	precompute := uint(8)
	if err = ckzg4844.LoadTrustedSetup(g1MonomialBytes, g1LagrangeBytes, g2MonomialBytes, precompute); err != nil {
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

// VerifyCellKZGProofBatch verifies a batch of KZG proofs for a set of cells.
func ckzgVerifyCellKZGProofBatch(commitments []Commitment, cellIndicies []uint64, cells []Cell, proofs []Proof) error {
	ckzgIniter.Do(ckzgInit)

	comts := make([]ckzg4844.Bytes48, len(commitments))
	for i, commitment := range commitments {
		comts[i] = (ckzg4844.Bytes48)(commitment)
	}

	cellInputs := make([]ckzg4844.Cell, len(cells))
	for i := range cells {
		cellInputs[i] = (ckzg4844.Cell)(cells[i])
	}

	proofsInput := make([]ckzg4844.Bytes48, len(proofs))
	for i, proof := range proofs {
		proofsInput[i] = (ckzg4844.Bytes48)(proof)
	}

	valid, err := ckzg4844.VerifyCellKZGProofBatch(comts, cellIndicies, cellInputs, proofsInput)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proof")
	}
	return nil
}

func ckzgComputeCells(blob *Blob) ([]Cell, error) {
	ckzgIniter.Do(ckzgInit)

	result, err := ckzg4844.ComputeCells((*ckzg4844.Blob)(blob))
	if err != nil {
		return nil, err
	}

	cells := make([]Cell, len(result))
	for i, cell := range result {
		cells[i] = (Cell)(cell)
	}
	return cells, nil
}
