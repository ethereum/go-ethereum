package legacypool

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
)

// Max returns the larger of x or y.
func max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

func TestCustomTxValidationEnforced(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := newTestBlockChain(eip1559Config, 10000000, statedb, new(event.Feed))

	txPoolConfig := DefaultConfig
	if txPoolConfig.CustomValidationEnabled {
		t.Fatalf("Custom validation should be disabled by default")
	}

	txPoolConfig.NoLocals = true
	pool := New(txPoolConfig, blockchain)
	pool.Init(new(big.Int).SetUint64(txPoolConfig.PriceLimit), blockchain.CurrentBlock(), makeAddressReserver())
	defer pool.Close()

	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000))

	tx := pricedTransaction(0, 100000, big.NewInt(3), key)
	pool.SetGasTip(big.NewInt(tx.GasPrice().Int64()))

	if err := pool.Add([]*types.Transaction{tx}, true, false)[0]; err != nil {
		t.Fatalf("TxValidation enforced in default config despite being disabled by default")
	}

	// Modifying local config to enable custom validation
	poolWithCustomValidationConfig := DefaultConfig
	poolWithCustomValidationConfig.CustomValidationEnabled = true
	poolWithCustomValidationConfig.NoLocals = true

	bannedKey, _ := crypto.GenerateKey()
	bannedTx := pricedTransaction(0, 990000, big.NewInt(4), bannedKey)

	bannedAddresses := []common.Address{crypto.PubkeyToAddress(bannedKey.PublicKey)}
	hypernativeCustomValidatorConfig := &txpool.CustomValidatorConfigHypernative{
		BannedAddresses: bannedAddresses,
	}
	poolWithCustomValidationConfig.CustomValidator = txpool.NewHypernativeValidator(hypernativeCustomValidatorConfig)

	poolWithCustomValidation := New(poolWithCustomValidationConfig, blockchain)
	poolWithCustomValidation.Init(new(big.Int).SetUint64(poolWithCustomValidationConfig.PriceLimit), blockchain.CurrentBlock(), makeAddressReserver())
	defer poolWithCustomValidation.Close()

	testAddBalance(pool, crypto.PubkeyToAddress(bannedKey.PublicKey), big.NewInt(1000000))
	testAddBalance(poolWithCustomValidation, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000))

	gasTip := min(tx.GasPrice().Int64(), bannedTx.GasPrice().Int64())
	poolWithCustomValidation.SetGasTip(big.NewInt(gasTip))

	if err := poolWithCustomValidation.Add([]*types.Transaction{tx}, true, false)[0]; err != nil {
		t.Fatalf("Custom TxValidation enforced wrongly")
	}

	if err := poolWithCustomValidation.Add([]*types.Transaction{bannedTx}, true, false)[0]; !errors.Is(err, txpool.ErrCustomValidationFailed) {
		t.Fatalf("Custom TxValidation not enforced for banned address")
	}
}
