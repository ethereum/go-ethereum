// Copyright 2026 The go-ethereum Authors
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

package vm

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ubtVanillaTestChainConfig is a post-merge chain with only Shanghai + UBT (Verkle /
// EIP-4762) enabled, matching the fork selection order in NewEVM so the Verkle
// instruction set is active (not Prague/Osaka/Amsterdam ahead of UBT in the switch).
var ubtVanillaTestChainConfig = func() *params.ChainConfig {
	ts := uint64(0)
	return &params.ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		Ethash:                  new(params.EthashConfig),
		ShanghaiTime:            &ts,
		UBTTime:                 &ts,
		BlobScheduleConfig: &params.BlobScheduleConfig{
			UBT: params.DefaultPragueBlobConfig,
		},
	}
}()

// TestContractCreateEIP4762InitGasOOGRevertsSnapshot checks that when EIP-4762 witness
// metering for contract-init cannot be fully paid, creation aborts with ErrOutOfGas and
// state changes after Snapshot (new contract account scaffolding) are rolled back.
//
// Without RevertToSnapshot on that path, create() would leave an empty contract shell
// at the creation address even though no initcode ran successfully.
func TestContractCreateEIP4762InitGasOOGRevertsSnapshot(t *testing.T) {
	var (
		statedb, _ = state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		caller     = common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		blockCtx   = BlockContext{
			CanTransfer: func(db StateDB, addr common.Address, amount *uint256.Int) bool {
				return db.GetBalance(addr).Cmp(amount) >= 0
			},
			Transfer: func(db StateDB, s, r common.Address, amount *uint256.Int, _ *params.Rules) {
				db.SubBalance(s, amount, tracing.BalanceChangeTransfer)
				db.AddBalance(r, amount, tracing.BalanceChangeTransfer)
			},
			BlockNumber: big.NewInt(0),
			Time:        0,
			Random:      &common.Hash{},
		}
	)
	statedb.CreateAccount(caller)
	statedb.AddBalance(caller, uint256.NewInt(1e18), tracing.BalanceIncreaseGenesisBalance)

	contractAddr := crypto.CreateAddress(caller, statedb.GetNonce(caller))

	// Derive a gas budget that passes ContractCreatePreCheckGas but leaves too little
	// regular gas for the subsequent ContractCreateInitGas witness charge.
	aeCalc := state.NewAccessEvents()
	precheckCharge := aeCalc.ContractCreatePreCheckGas(contractAddr, math.MaxUint64)
	paid, needed := aeCalc.ContractCreateInitGas(contractAddr, math.MaxUint64)
	if paid != needed {
		t.Fatalf("broken setup: init witness metering should settle with MaxUint64 (paid=%d needed=%d)", paid, needed)
	}
	verify := state.NewAccessEvents()
	verify.ContractCreatePreCheckGas(contractAddr, math.MaxUint64)
	partialPaid, partialNeed := verify.ContractCreateInitGas(contractAddr, paid-1)
	if partialPaid >= partialNeed {
		t.Fatalf("broken setup: need partial witness settle (paid=%d need=%d)", partialPaid, partialNeed)
	}
	gasBudget := precheckCharge + paid - 1

	evm := NewEVM(blockCtx, statedb, ubtVanillaTestChainConfig, Config{})
	evm.TxContext = TxContext{
		Origin:       caller,
		AccessEvents: state.NewAccessEvents(),
		GasPrice:     new(uint256.Int),
	}

	initCode := []byte{0x00} // STOP — successful deployment that returns empty code
	_, _, _, err := evm.Create(caller, initCode, NewGasBudget(gasBudget), uint256.NewInt(0))

	if err != ErrOutOfGas {
		t.Fatalf("expected ErrOutOfGas, got %v", err)
	}
	if statedb.Exist(contractAddr) {
		t.Fatal("creation address should not exist after OOG revert (contract shell must be rolled back)")
	}

	nonceAfter := statedb.GetNonce(caller)
	if nonceAfter != 1 {
		t.Fatalf("caller nonce should still increment once (create started), got %d want 1", nonceAfter)
	}
}
