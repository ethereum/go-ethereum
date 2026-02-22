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

package apitypes

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
)

func TestIsPrimitive(t *testing.T) {
	t.Parallel()
	// Expected positives
	for i, tc := range []string{
		"int24", "int24[]", "int[]", "int[2]", "uint88", "uint88[]", "uint", "uint[]", "uint[2]", "int256", "int256[]",
		"uint96", "uint96[]", "int96", "int96[]", "bytes17[]", "bytes17", "address[2]", "bool[4]", "string[5]", "bytes[2]",
		"bytes32", "bytes32[]", "bytes32[4]",
	} {
		if !isPrimitiveTypeValid(tc) {
			t.Errorf("test %d: expected '%v' to be a valid primitive", i, tc)
		}
	}
	// Expected negatives
	for i, tc := range []string{
		"int257", "int257[]", "uint88 ", "uint88 []", "uint257", "uint-1[]",
		"uint0", "uint0[]", "int95", "int95[]", "uint1", "uint1[]", "bytes33[]", "bytess",
	} {
		if isPrimitiveTypeValid(tc) {
			t.Errorf("test %d: expected '%v' to not be a valid primitive", i, tc)
		}
	}
}

func TestTxArgs(t *testing.T) {
	for i, tc := range []struct {
		data     []byte
		want     common.Hash
		wantType uint8
	}{
		{
			data:     []byte(`{"from":"0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425","accessList":[],"blobVersionedHashes":["0x010657f37554c781402a22917dee2f75def7ab966d7b770905398eba3c444014"],"chainId":"0x7","gas":"0x124f8","gasPrice":"0x693d4ca8","input":"0x","maxFeePerBlobGas":"0x3b9aca00","maxFeePerGas":"0x6fc23ac00","maxPriorityFeePerGas":"0x3b9aca00","nonce":"0x0","r":"0x2a922afc784d07e98012da29f2f37cae1f73eda78aa8805d3df6ee5dbb41ec1","s":"0x4f1f75ae6bcdf4970b4f305da1a15d8c5ddb21f555444beab77c9af2baab14","to":"0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425","type":"0x1","v":"0x0","value":"0x0","yParity":"0x0"}`),
			want:     common.HexToHash("0x7d53234acc11ac5b5948632c901a944694e228795782f511887d36fd76ff15c4"),
			wantType: types.BlobTxType,
		},
		{
			// on input, we don't read the type, but infer the type from the arguments present
			data:     []byte(`{"from":"0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425","accessList":[],"chainId":"0x7","gas":"0x124f8","gasPrice":"0x693d4ca8","input":"0x","maxFeePerBlobGas":"0x3b9aca00","maxFeePerGas":"0x6fc23ac00","maxPriorityFeePerGas":"0x3b9aca00","nonce":"0x0","r":"0x2a922afc784d07e98012da29f2f37cae1f73eda78aa8805d3df6ee5dbb41ec1","s":"0x4f1f75ae6bcdf4970b4f305da1a15d8c5ddb21f555444beab77c9af2baab14","to":"0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425","type":"0x12","v":"0x0","value":"0x0","yParity":"0x0"}`),
			want:     common.HexToHash("0x7919e2b0b9b543cb87a137b6ff66491ec7ae937cb88d3c29db4d9b28073dce53"),
			wantType: types.DynamicFeeTxType,
		},
	} {
		var txArgs SendTxArgs
		if err := json.Unmarshal(tc.data, &txArgs); err != nil {
			t.Fatal(err)
		}
		tx, err := txArgs.ToTransaction()
		if err != nil {
			t.Fatal(err)
		}
		if have := tx.Type(); have != tc.wantType {
			t.Errorf("test %d, have type %d, want type %d", i, have, tc.wantType)
		}
		if have := tx.Hash(); have != tc.want {
			t.Errorf("test %d: have %v, want %v", i, have, tc.want)
		}
		d2, err := json.Marshal(txArgs)
		if err != nil {
			t.Fatal(err)
		}
		var txArgs2 SendTxArgs
		if err := json.Unmarshal(d2, &txArgs2); err != nil {
			t.Fatal(err)
		}
		tx1, _ := txArgs.ToTransaction()
		tx2, _ := txArgs2.ToTransaction()
		if have, want := tx1.Hash(), tx2.Hash(); have != want {
			t.Errorf("test %d: have %v, want %v", i, have, want)
		}
	}
	/*
		End to end testing:

			$ go run ./cmd/clef --advanced --suppress-bootwarn

			$ go run ./cmd/geth --nodiscover --maxpeers 0 --signer /home/user/.clef/clef.ipc console

				> tx={"from":"0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425","to":"0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425","gas":"0x124f8","maxFeePerGas":"0x6fc23ac00","maxPriorityFeePerGas":"0x3b9aca00","value":"0x0","nonce":"0x0","input":"0x","accessList":[],"maxFeePerBlobGas":"0x3b9aca00","blobVersionedHashes":["0x010657f37554c781402a22917dee2f75def7ab966d7b770905398eba3c444014"]}
				> eth.signTransaction(tx)
	*/
}

func TestBlobTxs(t *testing.T) {
	blob := kzg4844.Blob{0x1}
	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := kzg4844.ComputeBlobProof(&blob, commitment)
	if err != nil {
		t.Fatal(err)
	}

	hash := kzg4844.CalcBlobHashV1(sha256.New(), &commitment)
	b := &types.BlobTx{
		ChainID:    uint256.NewInt(6),
		Nonce:      8,
		GasTipCap:  uint256.NewInt(500),
		GasFeeCap:  uint256.NewInt(600),
		Gas:        21000,
		BlobFeeCap: uint256.NewInt(700),
		BlobHashes: []common.Hash{hash},
		Value:      uint256.NewInt(100),
		Sidecar:    types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{blob}, []kzg4844.Commitment{commitment}, []kzg4844.Proof{proof}),
	}
	tx := types.NewTx(b)
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tx %v", string(data))
}

func TestType_IsArray(t *testing.T) {
	t.Parallel()
	// Expected positives
	for i, tc := range []Type{
		{
			Name: "type1",
			Type: "int24[]",
		},
		{
			Name: "type2",
			Type: "int24[2]",
		},
		{
			Name: "type3",
			Type: "int24[2][2][2]",
		},
	} {
		if !tc.isArray() {
			t.Errorf("test %d: expected '%v' to be an array", i, tc)
		}
	}
	// Expected negatives
	for i, tc := range []Type{
		{
			Name: "type1",
			Type: "int24",
		},
		{
			Name: "type2",
			Type: "uint88",
		},
		{
			Name: "type3",
			Type: "bytes32",
		},
	} {
		if tc.isArray() {
			t.Errorf("test %d: expected '%v' to not be an array", i, tc)
		}
	}
}

func TestType_TypeName(t *testing.T) {
	t.Parallel()

	for i, tc := range []struct {
		Input    Type
		Expected string
	}{
		{
			Input: Type{
				Name: "type1",
				Type: "int24[]",
			},
			Expected: "int24",
		},
		{
			Input: Type{
				Name: "type2",
				Type: "int26[2][2][2]",
			},
			Expected: "int26",
		},
		{
			Input: Type{
				Name: "type3",
				Type: "int24",
			},
			Expected: "int24",
		},
		{
			Input: Type{
				Name: "type4",
				Type: "uint88",
			},
			Expected: "uint88",
		},
		{
			Input: Type{
				Name: "type5",
				Type: "bytes32[2]",
			},
			Expected: "bytes32",
		},
	} {
		if tc.Input.typeName() != tc.Expected {
			t.Errorf("test %d: expected typeName value of '%v' but got '%v'", i, tc.Expected, tc.Input)
		}
	}
}

func TestValidateTxSidecar(t *testing.T) {
	t.Parallel()

	// Helper function to create a test blob and its commitment/proof
	createTestBlob := func() (kzg4844.Blob, kzg4844.Commitment, kzg4844.Proof, common.Hash) {
		b := make([]byte, 31)
		rand.Read(b)
		var blob kzg4844.Blob
		for i := range b {
			blob[i+1] = b[i]
		}
		commitment, err := kzg4844.BlobToCommitment(&blob)
		if err != nil {
			t.Fatal(err)
		}
		proof, err := kzg4844.ComputeBlobProof(&blob, commitment)
		if err != nil {
			t.Fatal(err)
		}
		hash := kzg4844.CalcBlobHashV1(sha256.New(), &commitment)
		return blob, commitment, proof, hash
	}

	// Helper function to create test cell proofs for v1 transactions
	createTestCellProofs := func(blob kzg4844.Blob) []kzg4844.Proof {
		cellProofs, err := kzg4844.ComputeCellProofs(&blob)
		if err != nil {
			t.Fatal(err)
		}
		return cellProofs
	}

	blob1, commitment1, proof1, hash1 := createTestBlob()
	blob2, commitment2, proof2, hash2 := createTestBlob()

	tests := []struct {
		name    string
		args    SendTxArgs
		wantErr bool
	}{
		{
			name:    "no blobs - should pass",
			args:    SendTxArgs{},
			wantErr: false,
		},
		{
			name: "valid blobs with commitments and proofs",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1, blob2},
				Commitments: []kzg4844.Commitment{commitment1, commitment2},
				Proofs:      []kzg4844.Proof{proof1, proof2},
				BlobHashes:  []common.Hash{hash1, hash2},
			},
			wantErr: false,
		},
		{
			name: "valid blobs without commitments/proofs - should generate them",
			args: SendTxArgs{
				Blobs: []kzg4844.Blob{blob1},
			},
			wantErr: false,
		},
		{
			name: "valid blobs with v1 cell proofs",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1},
				Commitments: []kzg4844.Commitment{commitment1},
				Proofs:      createTestCellProofs(blob1),
				BlobHashes:  []common.Hash{hash1},
			},
			wantErr: false,
		},
		{
			name: "blobs with v1 version flag - should generate cell proofs",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1},
				BlobVersion: types.BlobSidecarVersion1,
			},
			wantErr: false,
		},
		{
			name: "proofs provided but commitments not",
			args: SendTxArgs{
				Blobs:  []kzg4844.Blob{blob1},
				Proofs: []kzg4844.Proof{proof1},
			},
			wantErr: true,
		},
		{
			name: "commitments provided but proofs not",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1},
				Commitments: []kzg4844.Commitment{commitment1},
			},
			wantErr: true,
		},
		{
			name: "mismatch between blobs and commitments",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1, blob2},
				Commitments: []kzg4844.Commitment{commitment1}, // Only one commitment for two blobs
				Proofs:      []kzg4844.Proof{proof1},
			},
			wantErr: true,
		},
		{
			name: "mismatch between blobs and hashes",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1, blob2},
				Commitments: []kzg4844.Commitment{commitment1, commitment2},
				Proofs:      []kzg4844.Proof{proof1, proof2},
				BlobHashes:  []common.Hash{hash1}, // Only one hash for two blobs
			},
			wantErr: true,
		},
		{
			name: "wrong number of proofs",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1, blob2},
				Commitments: []kzg4844.Commitment{commitment1, commitment2},
				Proofs:      []kzg4844.Proof{proof1, proof2, proof1}, // 3 proofs for 2 blobs
			},
			wantErr: true,
		},
		{
			name: "invalid blob hash",
			args: SendTxArgs{
				Blobs:       []kzg4844.Blob{blob1},
				Commitments: []kzg4844.Commitment{commitment1},
				Proofs:      []kzg4844.Proof{proof1},
				BlobHashes:  []common.Hash{hash2}, // Wrong hash
			},
			wantErr: true,
		},
		{
			name: "invalid proof",
			args: SendTxArgs{
				BlobVersion: types.BlobSidecarVersion1,
				Blobs:       []kzg4844.Blob{blob1},
				Commitments: []kzg4844.Commitment{commitment1},
				Proofs:      []kzg4844.Proof{proof1, proof2}, // wrong proof
				BlobHashes:  []common.Hash{hash1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original test case
			args := tt.args
			err := args.validateTxSidecar()

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTxSidecar() expected error but got none")
					return
				}
			} else {
				if err != nil {
					t.Errorf("validateTxSidecar() unexpected error = %v", err)
				}

				// For successful cases, verify that commitments and proofs were generated if they weren't provided
				if len(args.Blobs) > 0 {
					if args.Commitments == nil || len(args.Commitments) != len(args.Blobs) {
						t.Errorf("validateTxSidecar() should have generated commitments")
					}
					if args.Proofs == nil || (len(args.Proofs) != len(args.Blobs) && len(args.Proofs) != len(args.Blobs)*kzg4844.CellProofsPerBlob) {
						t.Errorf("validateTxSidecar() should have generated proofs")
					}
					if args.BlobHashes == nil || len(args.BlobHashes) != len(args.Blobs) {
						t.Errorf("validateTxSidecar() should have generated blob hashes")
					}
				}
			}
		})
	}
}
