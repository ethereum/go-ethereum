package pulse

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
)

func TestReplaceDepositContract(t *testing.T) {
	// Init
	db := rawdb.NewMemoryDatabase()
	state, _ := state.New(common.Hash{}, state.NewDatabaseWithConfig(db, &trie.Config{Preimages: true}), nil)

	// Exec
	replaceDepositContract(state)

	// Verify
	balance := state.GetBalance(pulseDepositContractAddr)
	if balance.Cmp(common.Big0) != 0 {
		t.Errorf("Found unexpected deposit contract balance: %d", balance)
	}

	actualCode := state.GetCode(pulseDepositContractAddr)
	for i, b := range actualCode {
		if b != depositContractBytes[i] {
			t.Errorf("Invalid deposit contract code at index %d", i)
		}
	}

	// Verify Storage
	for i, store := range depositContractStorage {
		actualStorage := state.GetState(pulseDepositContractAddr, common.HexToHash(store[0]))
		expectedStorage := common.HexToHash(store[1])
		if actualStorage != expectedStorage {
			t.Errorf("Invalid storage entry %d, actual: %d, expected: %d", i, actualStorage, expectedStorage)
		} else {
			t.Log("Valid Storage entry")
		}
	}
}
