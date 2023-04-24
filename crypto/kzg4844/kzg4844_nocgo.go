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

//go:build nacl || js || !cgo || gofuzz

package kzg4844

import gokzg4844 "github.com/crate-crypto/go-kzg-4844"

// context is the crypto primitive pre-seeded with the trusted setup parameters.
var context *gokzg4844.Context

// initKZG initializes the KZG library with the provided trusted setup.
func initKZG(params *gokzg4844.JSONTrustedSetup) error {
	err := gokzg4844.CheckTrustedSetupIsWellFormed(params)
	if err != nil {
		return err
	}
	context, err = gokzg4844.NewContext4096(params)
	if err != nil {
		return err
	}
	return nil
}

// BlobToCommitment creates a small commitment out of a data blob.
func BlobToCommitment(blob Blob) (Commitment, error) {
	commitment, err := context.BlobToKZGCommitment((gokzg4844.Blob)(blob))
	if err != nil {
		return Commitment{}, err
	}
	return (Commitment)(commitment), nil
}

// ComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func ComputeProof(blob Blob, point Point) (Proof, Claim, error) {
	proof, claim, err := context.ComputeKZGProof((gokzg4844.Blob)(blob), (gokzg4844.Scalar)(point))
	if err != nil {
		return Proof{}, Claim{}, err
	}
	return (Proof)(proof), (Claim)(claim), nil
}

// VerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
func VerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	return context.VerifyKZGProof((gokzg4844.KZGCommitment)(commitment), (gokzg4844.Scalar)(point), (gokzg4844.Scalar)(claim), (gokzg4844.KZGProof)(proof))
}

// ComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func ComputeBlobProof(blob Blob, commitment Commitment) (Proof, error) {
	proof, err := context.ComputeBlobKZGProof((gokzg4844.Blob)(blob), (gokzg4844.KZGCommitment)(commitment))
	if err != nil {
		return Proof{}, err
	}
	return (Proof)(proof), nil
}

// VerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func VerifyBlobProof(blob Blob, commitment Commitment, proof Proof) error {
	return context.VerifyBlobKZGProof((gokzg4844.Blob)(blob), (gokzg4844.KZGCommitment)(commitment), (gokzg4844.KZGProof)(proof))
}
