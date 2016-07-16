package core

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Test if genesis block is deployed with correct state.
func TestWriteGenesisBlockForTesting(t *testing.T) {
	type Expected struct {
		code    []byte
		balance *big.Int
		storage map[common.Hash]common.Hash
	}

	stor := map[common.Hash]common.Hash{
		common.HexToHash("0x00"):                                                               common.HexToHash("0x01"),
		common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"): common.HexToHash("abcdef"),
	}

	tests := []struct {
		account  GenesisAccount
		expected Expected
	}{
		{
			account: GenesisAccount{
				Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
				Balance: big.NewInt(11111),
			},
			expected: Expected{
				balance: big.NewInt(11111),
			},
		},
		{
			account: GenesisAccount{
				Address: common.HexToAddress("0x0000000000000000000000000000000000000002"),
				Balance: big.NewInt(22222),
				Code:    common.FromHex("60606040526008565b00"), // contract Storage { uint a; mapping(address => uint) map; }
				Storage: stor,
			},
			expected: Expected{
				code:    common.FromHex("60606040526008565b00"),
				balance: big.NewInt(22222),
				storage: stor,
			},
		},
	}

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		t.Fatal(err)
	}

	var accounts []GenesisAccount
	for _, test := range tests {
		accounts = append(accounts, test.account)
	}
	block := WriteGenesisBlockForTesting(db, accounts...)

	s, err := state.New(block.Root(), db)
	if err != nil {
		t.Fatal(err)
	}

	for i, test := range tests {
		if test.expected.balance != nil {
			balance := s.GetBalance(test.account.Address)
			if test.expected.balance.Cmp(balance) != 0 {
				t.Errorf("invalid balance for test %d: want %d, got %d", i, test.expected.balance, balance)
			}
		}
		if len(test.expected.code) > 0 {
			code := s.GetCode(test.account.Address)
			if !bytes.Equal(test.expected.code, code) {
				t.Errorf("invalid code stored for test %d: want %x, got %x", i, test.expected.code, code)
			}
		}
		for k, v := range test.expected.storage {
			val := s.GetState(test.account.Address, k)
			if v != val {
				t.Errorf("invalid storage for test %d: want %x, got %x", i, v, val)
			}
		}
	}
}
