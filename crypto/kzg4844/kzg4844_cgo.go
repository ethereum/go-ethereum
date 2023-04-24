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

//go:build !nacl && !js && cgo && !gofuzz

package kzg4844

import (
	"errors"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	ckzg4844 "github.com/ethereum/c-kzg-4844/bindings/go"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// initKZG initializes the KZG library with the provided trusted setup.
func initKZG(params *gokzg4844.JSONTrustedSetup) error {
	err := gokzg4844.CheckTrustedSetupIsWellFormed(params)
	if err != nil {
		return err
	}
	g1s := make([]byte, len(params.SetupG1)*len(params.SetupG1[0])/2)
	for i, g1 := range params.SetupG1 {
		copy(g1s[i*len(g1)/2:], hexutil.MustDecode("0x"+g1))
	}
	g2s := make([]byte, len(params.SetupG2)*len(params.SetupG2[0])/2)
	for i, g2 := range params.SetupG2 {
		copy(g2s[i*len(g2)/2:], hexutil.MustDecode("0x"+g2))
	}
	if err := ckzg4844.LoadTrustedSetup(g1s, g2s); err != nil {
		return err
	}
	return nil
}

// BlobToCommitment creates a small commitment out of a data blob.
func BlobToCommitment(blob Blob) (Commitment, error) {
	commitment, err := ckzg4844.BlobToKZGCommitment((ckzg4844.Blob)(blob))
	if err != nil {
		return Commitment{}, err
	}
	return (Commitment)(commitment), nil
}

// ComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func ComputeProof(blob Blob, point Point) (Proof, Claim, error) {
	proof, claim, err := ckzg4844.ComputeKZGProof((ckzg4844.Blob)(blob), (ckzg4844.Bytes32)(point))
	if err != nil {
		return Proof{}, Claim{}, err
	}
	return (Proof)(proof), (Claim)(claim), nil
}

// VerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
func VerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	valid, err := ckzg4844.VerifyKZGProof((ckzg4844.Bytes48)(commitment), (ckzg4844.Bytes32)(point), (ckzg4844.Bytes32)(claim), (ckzg4844.Bytes48)(proof))
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proof")
	}
	return nil
}

// ComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func ComputeBlobProof(blob Blob, commitment Commitment) (Proof, error) {
	proof, err := ckzg4844.ComputeBlobKZGProof((ckzg4844.Blob)(blob), (ckzg4844.Bytes48)(commitment))
	if err != nil {
		return Proof{}, err
	}
	return (Proof)(proof), nil
}

// VerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func VerifyBlobProof(blob Blob, commitment Commitment, proof Proof) error {
	valid, err := ckzg4844.VerifyBlobKZGProof((ckzg4844.Blob)(blob), (ckzg4844.Bytes48)(commitment), (ckzg4844.Bytes48)(proof))
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid proof")
	}
	return nil
}
