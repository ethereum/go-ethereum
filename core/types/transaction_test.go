// Copyright 2014 The go-ethereum Authors
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

package types

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// The values in those tests are from the Transaction Tests
// at github.com/ethereum/tests.
var (
	testAddr = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b")

	emptyTx = NewTransaction(
		0,
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), 0, big.NewInt(0),
		nil,
	)

	rightvrsTx, _ = NewTransaction(
		3,
		testAddr,
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
	).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	emptyEip2718Tx = NewTx(&AccessListTx{
		ChainID:  big.NewInt(1),
		Nonce:    3,
		To:       &testAddr,
		Value:    big.NewInt(10),
		Gas:      25000,
		GasPrice: big.NewInt(1),
		Data:     common.FromHex("5544"),
	})

	signedEip2718Tx, _ = emptyEip2718Tx.WithSignature(
		NewEIP2930Signer(big.NewInt(1)),
		common.Hex2Bytes("c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b266032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d3752101"),
	)
)

func TestDecodeEmptyTypedTx(t *testing.T) {
	input := []byte{0x80}
	var tx Transaction
	err := rlp.DecodeBytes(input, &tx)
	if err != errShortTypedTx {
		t.Fatal("wrong error:", err)
	}
}

func TestTransactionSigHash(t *testing.T) {
	var homestead HomesteadSigner
	if homestead.Hash(emptyTx) != common.HexToHash("c775b99e7ad12f50d819fcd602390467e28141316969f4b57f0626f74fe3b386") {
		t.Errorf("empty transaction hash mismatch, got %x", emptyTx.Hash())
	}
	if homestead.Hash(rightvrsTx) != common.HexToHash("fe7a79529ed5f7c3375d06b26b186a8644e0e16c373d7a12be41c62d6042b77a") {
		t.Errorf("RightVRS transaction hash mismatch, got %x", rightvrsTx.Hash())
	}
}

func TestTransactionEncode(t *testing.T) {
	txb, err := rlp.EncodeToBytes(rightvrsTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	should := common.FromHex("f86103018207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a8255441ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
	if !bytes.Equal(txb, should) {
		t.Errorf("encoded RLP mismatch, got %x", txb)
	}
}

func TestEIP2718TransactionSigHash(t *testing.T) {
	s := NewEIP2930Signer(big.NewInt(1))
	if s.Hash(emptyEip2718Tx) != common.HexToHash("49b486f0ec0a60dfbbca2d30cb07c9e8ffb2a2ff41f29a1ab6737475f6ff69f3") {
		t.Errorf("empty EIP-2718 transaction hash mismatch, got %x", s.Hash(emptyEip2718Tx))
	}
	if s.Hash(signedEip2718Tx) != common.HexToHash("49b486f0ec0a60dfbbca2d30cb07c9e8ffb2a2ff41f29a1ab6737475f6ff69f3") {
		t.Errorf("signed EIP-2718 transaction hash mismatch, got %x", s.Hash(signedEip2718Tx))
	}
}

// This test checks signature operations on access list transactions.
func TestEIP2930Signer(t *testing.T) {
	var (
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		keyAddr = crypto.PubkeyToAddress(key.PublicKey)
		signer1 = NewEIP2930Signer(big.NewInt(1))
		signer2 = NewEIP2930Signer(big.NewInt(2))
		tx0     = NewTx(&AccessListTx{Nonce: 1})
		tx1     = NewTx(&AccessListTx{ChainID: big.NewInt(1), Nonce: 1})
		tx2, _  = SignNewTx(key, signer2, &AccessListTx{ChainID: big.NewInt(2), Nonce: 1})
	)

	tests := []struct {
		tx             *Transaction
		signer         Signer
		wantSignerHash common.Hash
		wantSenderErr  error
		wantSignErr    error
		wantHash       common.Hash // after signing
	}{
		{
			tx:             tx0,
			signer:         signer1,
			wantSignerHash: common.HexToHash("846ad7672f2a3a40c1f959cd4a8ad21786d620077084d84c8d7c077714caa139"),
			wantSenderErr:  ErrInvalidChainId,
			wantHash:       common.HexToHash("1ccd12d8bbdb96ea391af49a35ab641e219b2dd638dea375f2bc94dd290f2549"),
		},
		{
			tx:             tx1,
			signer:         signer1,
			wantSenderErr:  ErrInvalidSig,
			wantSignerHash: common.HexToHash("846ad7672f2a3a40c1f959cd4a8ad21786d620077084d84c8d7c077714caa139"),
			wantHash:       common.HexToHash("1ccd12d8bbdb96ea391af49a35ab641e219b2dd638dea375f2bc94dd290f2549"),
		},
		{
			// This checks what happens when trying to sign an unsigned tx for the wrong chain.
			tx:             tx1,
			signer:         signer2,
			wantSenderErr:  ErrInvalidChainId,
			wantSignerHash: common.HexToHash("367967247499343401261d718ed5aa4c9486583e4d89251afce47f4a33c33362"),
			wantSignErr:    ErrInvalidChainId,
		},
		{
			// This checks what happens when trying to re-sign a signed tx for the wrong chain.
			tx:             tx2,
			signer:         signer1,
			wantSenderErr:  ErrInvalidChainId,
			wantSignerHash: common.HexToHash("846ad7672f2a3a40c1f959cd4a8ad21786d620077084d84c8d7c077714caa139"),
			wantSignErr:    ErrInvalidChainId,
		},
	}

	for i, test := range tests {
		sigHash := test.signer.Hash(test.tx)
		if sigHash != test.wantSignerHash {
			t.Errorf("test %d: wrong sig hash: got %x, want %x", i, sigHash, test.wantSignerHash)
		}
		sender, err := Sender(test.signer, test.tx)
		if !errors.Is(err, test.wantSenderErr) {
			t.Errorf("test %d: wrong Sender error %q", i, err)
		}
		if err == nil && sender != keyAddr {
			t.Errorf("test %d: wrong sender address %x", i, sender)
		}
		signedTx, err := SignTx(test.tx, test.signer, key)
		if !errors.Is(err, test.wantSignErr) {
			t.Fatalf("test %d: wrong SignTx error %q", i, err)
		}
		if signedTx != nil {
			if signedTx.Hash() != test.wantHash {
				t.Errorf("test %d: wrong tx hash after signing: got %x, want %x", i, signedTx.Hash(), test.wantHash)
			}
		}
	}
}

func TestEIP2718TransactionEncode(t *testing.T) {
	// RLP representation
	{
		have, err := rlp.EncodeToBytes(signedEip2718Tx)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}
		want := common.FromHex("b86601f8630103018261a894b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a825544c001a0c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b2660a032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d37521")
		if !bytes.Equal(have, want) {
			t.Errorf("encoded RLP mismatch, got %x", have)
		}
	}
	// Binary representation
	{
		have, err := signedEip2718Tx.MarshalBinary()
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}
		want := common.FromHex("01f8630103018261a894b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a825544c001a0c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b2660a032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d37521")
		if !bytes.Equal(have, want) {
			t.Errorf("encoded RLP mismatch, got %x", have)
		}
	}
}

func decodeTx(data []byte) (*Transaction, error) {
	var tx Transaction
	t, err := &tx, rlp.DecodeBytes(data, &tx)
	return t, err
}

func defaultTestKey() (*ecdsa.PrivateKey, common.Address) {
	key, _ := crypto.HexToECDSA("45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

func TestRecipientEmpty(t *testing.T) {
	_, addr := defaultTestKey()
	tx, err := decodeTx(common.Hex2Bytes("f8498080808080011ca09b16de9d5bdee2cf56c28d16275a4da68cd30273e2525f3959f5d62557489921a0372ebd8fb3345f7db7b5a86d42e24d36e983e259b0664ceb8c227ec9af572f3d"))
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}
}

func TestRecipientNormal(t *testing.T) {
	_, addr := defaultTestKey()

	tx, err := decodeTx(common.Hex2Bytes("f85d80808094000000000000000000000000000000000000000080011ca0527c0d8f5c63f7b9f41324a7c8a563ee1190bcbf0dac8ab446291bdbf32f5c79a0552c4ef0a09a04395074dab9ed34d3fbfb843c2f2546cc30fe89ec143ca94ca6"))
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(HomesteadSigner{}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != from {
		t.Fatal("derived address doesn't match")
	}
}

// TestTransactionCoding tests serializing/de-serializing to/from rlp and JSON.
func TestTransactionCoding(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	var (
		signer    = NewEIP2930Signer(common.Big1)
		addr      = common.HexToAddress("0x0000000000000000000000000000000000000001")
		recipient = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
		accesses  = AccessList{{Address: addr, StorageKeys: []common.Hash{{0}}}}
	)
	for i := uint64(0); i < 500; i++ {
		var txdata TxData
		switch i % 5 {
		case 0:
			// Legacy tx.
			txdata = &LegacyTx{
				Nonce:    i,
				To:       &recipient,
				Gas:      1,
				GasPrice: big.NewInt(2),
				Data:     []byte("abcdef"),
			}
		case 1:
			// Legacy tx contract creation.
			txdata = &LegacyTx{
				Nonce:    i,
				Gas:      1,
				GasPrice: big.NewInt(2),
				Data:     []byte("abcdef"),
			}
		case 2:
			// Tx with non-zero access list.
			txdata = &AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      i,
				To:         &recipient,
				Gas:        123457,
				GasPrice:   big.NewInt(10),
				AccessList: accesses,
				Data:       []byte("abcdef"),
			}
		case 3:
			// Tx with empty access list.
			txdata = &AccessListTx{
				ChainID:  big.NewInt(1),
				Nonce:    i,
				To:       &recipient,
				Gas:      123457,
				GasPrice: big.NewInt(10),
				Data:     []byte("abcdef"),
			}
		case 4:
			// Contract creation with access list.
			txdata = &AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      i,
				Gas:        123457,
				GasPrice:   big.NewInt(10),
				AccessList: accesses,
			}
		}
		tx, err := SignNewTx(key, signer, txdata)
		if err != nil {
			t.Fatalf("could not sign transaction: %v", err)
		}
		// RLP
		parsedTx, err := encodeDecodeBinary(tx)
		if err != nil {
			t.Fatal(err)
		}
		if err := assertEqual(parsedTx, tx); err != nil {
			t.Fatal(err)
		}

		// JSON
		parsedTx, err = encodeDecodeJSON(tx)
		if err != nil {
			t.Fatal(err)
		}
		if err := assertEqual(parsedTx, tx); err != nil {
			t.Fatal(err)
		}
	}
}

func encodeDecodeJSON(tx *Transaction) (*Transaction, error) {
	data, err := json.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed: %v", err)
	}
	var parsedTx = &Transaction{}
	if err := json.Unmarshal(data, &parsedTx); err != nil {
		return nil, fmt.Errorf("json decoding failed: %v", err)
	}
	return parsedTx, nil
}

func encodeDecodeBinary(tx *Transaction) (*Transaction, error) {
	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("rlp encoding failed: %v", err)
	}
	var parsedTx = &Transaction{}
	if err := parsedTx.UnmarshalBinary(data); err != nil {
		return nil, fmt.Errorf("rlp decoding failed: %v", err)
	}
	return parsedTx, nil
}

func assertEqual(orig *Transaction, cpy *Transaction) error {
	// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
	if want, got := orig.Hash(), cpy.Hash(); want != got {
		return fmt.Errorf("parsed tx differs from original tx, want %v, got %v", want, got)
	}
	if want, got := orig.ChainId(), cpy.ChainId(); want.Cmp(got) != 0 {
		return fmt.Errorf("invalid chain id, want %d, got %d", want, got)
	}
	if orig.AccessList() != nil {
		if !reflect.DeepEqual(orig.AccessList(), cpy.AccessList()) {
			return errors.New("access list wrong!")
		}
	}
	return nil
}

func TestTransactionSizes(t *testing.T) {
	signer := NewLondonSigner(big.NewInt(123))
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	to := common.HexToAddress("0x01")
	for i, txdata := range []TxData{
		&AccessListTx{
			ChainID:  big.NewInt(123),
			Nonce:    0,
			To:       nil,
			Value:    big.NewInt(1000),
			Gas:      21000,
			GasPrice: big.NewInt(100000),
		},
		&LegacyTx{
			Nonce:    1,
			GasPrice: big.NewInt(500),
			Gas:      1000000,
			To:       &to,
			Value:    big.NewInt(1),
		},
		&AccessListTx{
			ChainID:  big.NewInt(123),
			Nonce:    1,
			GasPrice: big.NewInt(500),
			Gas:      1000000,
			To:       &to,
			Value:    big.NewInt(1),
			AccessList: AccessList{
				AccessTuple{
					Address:     common.HexToAddress("0x01"),
					StorageKeys: []common.Hash{common.HexToHash("0x01")},
				}},
		},
		&DynamicFeeTx{
			ChainID:   big.NewInt(123),
			Nonce:     1,
			Gas:       1000000,
			To:        &to,
			Value:     big.NewInt(1),
			GasTipCap: big.NewInt(500),
			GasFeeCap: big.NewInt(500),
		},
	} {
		tx, err := SignNewTx(key, signer, txdata)
		if err != nil {
			t.Fatalf("test %d: %v", i, err)
		}
		bin, _ := tx.MarshalBinary()

		// Check initial calc
		if have, want := int(tx.Size()), len(bin); have != want {
			t.Errorf("test %d: size wrong, have %d want %d", i, have, want)
		}
		// Check cached version too
		if have, want := int(tx.Size()), len(bin); have != want {
			t.Errorf("test %d: (cached) size wrong, have %d want %d", i, have, want)
		}
		// Check unmarshalled version too
		utx := new(Transaction)
		if err := utx.UnmarshalBinary(bin); err != nil {
			t.Fatalf("test %d: failed to unmarshal tx: %v", i, err)
		}
		if have, want := int(utx.Size()), len(bin); have != want {
			t.Errorf("test %d: (unmarshalled) size wrong, have %d want %d", i, have, want)
		}
	}
}

func TestYParityJSONUnmarshalling(t *testing.T) {
	baseJson := map[string]interface{}{
		// type is filled in by the test
		"chainId":              "0x7",
		"nonce":                "0x0",
		"to":                   "0x1b442286e32ddcaa6e2570ce9ed85f4b4fc87425",
		"gas":                  "0x124f8",
		"gasPrice":             "0x693d4ca8",
		"maxPriorityFeePerGas": "0x3b9aca00",
		"maxFeePerGas":         "0x6fc23ac00",
		"maxFeePerBlobGas":     "0x3b9aca00",
		"value":                "0x0",
		"input":                "0x",
		"accessList":           []interface{}{},
		"blobVersionedHashes": []string{
			"0x010657f37554c781402a22917dee2f75def7ab966d7b770905398eba3c444014",
		},

		// v and yParity are filled in by the test
		"r": "0x2a922afc784d07e98012da29f2f37cae1f73eda78aa8805d3df6ee5dbb41ec1",
		"s": "0x4f1f75ae6bcdf4970b4f305da1a15d8c5ddb21f555444beab77c9af2baab14",
	}

	tests := []struct {
		name    string
		v       string
		yParity string
		wantErr error
	}{
		// Valid v and yParity
		{"valid v and yParity, 0x0", "0x0", "0x0", nil},
		{"valid v and yParity, 0x1", "0x1", "0x1", nil},

		// Valid v, missing yParity
		{"valid v, missing yParity, 0x0", "0x0", "", nil},
		{"valid v, missing yParity, 0x1", "0x1", "", nil},

		// Valid yParity, missing v
		{"valid yParity, missing v, 0x0", "", "0x0", nil},
		{"valid yParity, missing v, 0x1", "", "0x1", nil},

		// Invalid yParity
		{"invalid yParity, 0x2", "", "0x2", errInvalidYParity},

		// Conflicting v and yParity
		{"conflicting v and yParity", "0x1", "0x0", errVYParityMismatch},

		// Missing v and yParity
		{"missing v and yParity", "", "", errVYParityMissing},
	}

	// Run for all types that accept yParity
	t.Parallel()
	for _, txType := range []uint64{
		AccessListTxType,
		DynamicFeeTxType,
		BlobTxType,
	} {
		txType := txType
		for _, test := range tests {
			test := test
			t.Run(fmt.Sprintf("txType=%d: %s", txType, test.name), func(t *testing.T) {
				// Copy the base json
				testJson := make(map[string]interface{})
				for k, v := range baseJson {
					testJson[k] = v
				}

				// Set v, yParity and type
				if test.v != "" {
					testJson["v"] = test.v
				}
				if test.yParity != "" {
					testJson["yParity"] = test.yParity
				}
				testJson["type"] = fmt.Sprintf("0x%x", txType)

				// Marshal the JSON
				jsonBytes, err := json.Marshal(testJson)
				if err != nil {
					t.Fatal(err)
				}

				// Unmarshal the tx
				var tx Transaction
				err = tx.UnmarshalJSON(jsonBytes)
				if err != test.wantErr {
					t.Fatalf("wrong error: got %v, want %v", err, test.wantErr)
				}
			})
		}
	}
}
