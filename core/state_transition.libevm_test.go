// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.
package core_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/libevm/hookstest"
	"github.com/ava-labs/libevm/params"
)

func TestCanExecuteTransaction(t *testing.T) {
	rng := ethtest.NewPseudoRand(42)
	account := rng.Address()
	slot := rng.Hash()

	makeErr := func(from common.Address, to *common.Address, val common.Hash) error {
		return fmt.Errorf("From: %v To: %v State: %v", from, to, val)
	}
	hooks := &hookstest.Stub{
		CanExecuteTransactionFn: func(from common.Address, to *common.Address, s libevm.StateReader) error {
			return makeErr(from, to, s.GetState(account, slot))
		},
	}
	hooks.Register(t)

	value := rng.Hash()

	state, evm := ethtest.NewZeroEVM(t)
	state.SetState(account, slot, value)
	msg := &core.Message{
		From: rng.Address(),
		To:   rng.AddressPtr(),
	}
	_, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(30e6))
	require.EqualError(t, err, makeErr(msg.From, msg.To, value).Error())
}

func TestMinimumGasConsumption(t *testing.T) {
	// All transactions will be basic transfers so consume [params.TxGas] by
	// default.
	tests := []struct {
		name           string
		gasLimit       uint64
		refund         uint64
		minConsumption uint64
		wantUsed       uint64
	}{
		{
			name:           "consume_extra",
			gasLimit:       1e6,
			minConsumption: 5e5,
			wantUsed:       5e5,
		},
		{
			name:           "consume_extra",
			gasLimit:       1e6,
			minConsumption: 4e5,
			wantUsed:       4e5,
		},
		{
			name:           "no_extra_consumption",
			gasLimit:       50_000,
			minConsumption: params.TxGas - 1,
			wantUsed:       params.TxGas,
		},
		{
			name:           "zero_min",
			gasLimit:       50_000,
			minConsumption: 0,
			wantUsed:       params.TxGas,
		},
		{
			name:           "consume_extra_by_one",
			gasLimit:       1e6,
			minConsumption: params.TxGas + 1,
			wantUsed:       params.TxGas + 1,
		},
		{
			name:           "min_capped_at_limit",
			gasLimit:       1e6,
			minConsumption: 2e6,
			wantUsed:       1e6,
		},
		{
			// Although this doesn't test minimum consumption, it demonstrates
			// the expected outcome for comparison with the next test.
			name:     "refund_without_min_consumption",
			gasLimit: 1e6,
			refund:   1,
			wantUsed: params.TxGas - 1,
		},
		{
			name:           "refund_with_min_consumption",
			gasLimit:       1e6,
			refund:         1,
			minConsumption: params.TxGas,
			wantUsed:       params.TxGas,
		},
	}

	// Very low gas price so we can calculate the expected balance in a uint64,
	// but not 1 otherwise tests would pass without multiplying extra
	// consumption by the price.
	const gasPrice = 3

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := &hookstest.Stub{
				MinimumGasConsumptionFn: func(limit uint64) uint64 {
					require.Equal(t, tt.gasLimit, limit)
					return tt.minConsumption
				},
			}
			hooks.Register(t)

			key, err := crypto.GenerateKey()
			require.NoError(t, err, "libevm/crypto.GenerateKey()")

			stateDB, evm := ethtest.NewZeroEVM(t)
			signer := types.LatestSigner(evm.ChainConfig())
			tx := types.MustSignNewTx(
				key, signer,
				&types.LegacyTx{
					GasPrice: big.NewInt(gasPrice),
					Gas:      tt.gasLimit,
					To:       &common.Address{},
					Value:    big.NewInt(0),
				},
			)

			const startingBalance = 10 * params.Ether
			from := crypto.PubkeyToAddress(key.PublicKey)
			stateDB.SetNonce(from, 0)
			stateDB.SetBalance(from, uint256.NewInt(startingBalance))
			stateDB.AddRefund(tt.refund)

			var (
				// Both variables are passed as pointers to
				// [core.ApplyTransaction], which will modify them.
				gotUsed uint64
				gotPool = core.GasPool(1e9)
			)
			wantPool := gotPool - core.GasPool(tt.wantUsed)

			receipt, err := core.ApplyTransaction(
				evm.ChainConfig(), nil, &common.Address{}, &gotPool, stateDB,
				&types.Header{
					BaseFee: big.NewInt(gasPrice),
					// Required but irrelevant fields
					Number:     big.NewInt(0),
					Difficulty: big.NewInt(0),
				},
				tx, &gotUsed, vm.Config{},
			)
			require.NoError(t, err, "core.ApplyTransaction(...)")

			for desc, got := range map[string]uint64{
				"receipt.GasUsed":                                  receipt.GasUsed,
				"receipt.CumulativeGasUsed":                        receipt.CumulativeGasUsed,
				"core.ApplyTransaction(..., usedGas *uint64, ...)": gotUsed,
			} {
				if got != tt.wantUsed {
					t.Errorf("%s got %d; want %d", desc, got, tt.wantUsed)
				}
			}
			if gotPool != wantPool {
				t.Errorf("After core.ApplyMessage(..., *%T); got %[1]T = %[1]d; want %d", gotPool, wantPool)
			}

			wantBalance := uint256.NewInt(startingBalance - tt.wantUsed*gasPrice)
			if got := stateDB.GetBalance(from); !got.Eq(wantBalance) {
				t.Errorf("got remaining balance %d; want %d", got, wantBalance)
			}
		})
	}
}
