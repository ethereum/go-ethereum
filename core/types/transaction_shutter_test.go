package types

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func assertAddress(t *testing.T, signer Signer, expected common.Address, tx *Transaction) {
	recovered, err := signer.Sender(tx)
	if err != nil {
		assertNilErr(t, err, "error while recovering address")
	}
	if recovered != expected {
		t.Fatal("recovered sender mismatch")
	}
}

func assertNilErr(t *testing.T, err error, message string) {
	if err != nil {
		if message == "" {
			t.Fatal(err)
		} else {
			t.Fatalf("%s: %v", message, err)
		}
	}

}

// TestTransactionCoding tests serializing/de-serializing to/from rlp and JSON.
func TestShutterTransactionCoding(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	var (
		signer        = NewLondonSigner(common.Big1)
		addr          = crypto.PubkeyToAddress(key.PublicKey)
		recipient     = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
		accesses      = AccessList{{Address: addr, StorageKeys: []common.Hash{{0}}}}
		decryptionKey = bytes.Repeat([]byte("x"), 32)
	)
	nonce := uint64(0)

	for j := uint64(0); j < 10; j++ {
		now := time.Now().Unix()

		transactions := make([]*Transaction, 0)
		transactionsBytes := make([][]byte, 0)
		for i := uint64(0); i < 20; i++ {
			var txdata TxData
			switch i % 2 {
			case 0:
				txdata = &ShutterTx{
					ChainID:          big.NewInt(1),
					Nonce:            nonce,
					BatchIndex:       j,
					Gas:              123457,
					GasTipCap:        big.NewInt(10),
					GasFeeCap:        big.NewInt(10),
					EncryptedPayload: []byte("foo"),
				}
			case 1:
				txdata = &DynamicFeeTx{
					ChainID:    big.NewInt(1),
					Nonce:      nonce,
					To:         &recipient,
					Gas:        123457,
					GasTipCap:  big.NewInt(10),
					GasFeeCap:  big.NewInt(10),
					AccessList: accesses,
					Data:       []byte("abcdef"),
				}
			}
			tx, err := SignNewTx(key, signer, txdata)
			assertNilErr(t, err, "can't sign tx")

			// RLP
			data, err := tx.MarshalBinary()
			assertNilErr(t, err, "can't marshal tx to binary")
			transactionsBytes = append(transactionsBytes, data)
			transactions = append(transactions, tx)
			nonce++
		}
		txdata := &BatchTx{
			ChainID:       big.NewInt(1),
			DecryptionKey: decryptionKey,
			BatchIndex:    j,
			L1BlockNumber: big.NewInt(42),
			Timestamp:     big.NewInt(now),
			Transactions:  transactionsBytes,
		}
		tx, err := SignNewTx(key, signer, txdata)
		assertNilErr(t, err, "can't sign tx")

		// check the outer transaction equality without the nested tx lists
		parsedTx, err := encodeDecodeBinary(tx)
		assertNilErr(t, err, "encoded/decoded tx not the same as initial tx")
		assertEqual(parsedTx, tx)

		// check signing
		assertAddress(t, signer, addr, parsedTx)

		// Now check equality of nested tx's
		for i, data := range parsedTx.Transactions() {
			var parsedTx = &Transaction{}
			err := parsedTx.UnmarshalBinary(data)
			assertNilErr(t, err, "rlp decoding of nested transaction failed")
			tx := transactions[i]
			assertEqual(parsedTx, tx)
			// check signing
			if recovered, _ := signer.Sender(parsedTx); recovered != addr {
				t.Fatal("recovered sender mismatch")
			}
			// check signing
			assertAddress(t, signer, addr, parsedTx)
		}
	}

}
