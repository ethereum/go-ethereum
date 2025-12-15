// Copyright 2025 the libevm authors.
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

package vm_test

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/params"
)

type preprocessingCharger struct {
	vm.NOOPHooks
	charge map[common.Hash]uint64
}

var errUnknownTx = errors.New("unknown tx")

func (p preprocessingCharger) PreprocessingGasCharge(tx common.Hash) (uint64, error) {
	c, ok := p.charge[tx]
	if !ok {
		return 0, fmt.Errorf("%w: %v", errUnknownTx, tx)
	}
	return c, nil
}

func TestChargePreprocessingGas(t *testing.T) {
	tests := []struct {
		name                   string
		to                     *common.Address
		charge                 uint64
		skipChargeRegistration bool
		txGas                  uint64
		wantVMErr              error
		wantGasUsed            uint64
	}{
		{
			name:        "standard create",
			to:          nil,
			txGas:       params.TxGas + params.CreateGas,
			wantGasUsed: params.TxGas + params.CreateGas,
		},
		{
			name:        "create with extra charge",
			to:          nil,
			charge:      1234,
			txGas:       params.TxGas + params.CreateGas + 2000,
			wantGasUsed: params.TxGas + params.CreateGas + 1234,
		},
		{
			name:        "standard call",
			to:          &common.Address{},
			txGas:       params.TxGas,
			wantGasUsed: params.TxGas,
		},
		{
			name:        "out of gas",
			to:          &common.Address{},
			charge:      1000,
			txGas:       params.TxGas + 999,
			wantGasUsed: params.TxGas + 999,
			wantVMErr:   vm.ErrOutOfGas,
		},
		{
			name:        "call with extra charge",
			to:          &common.Address{},
			charge:      13579,
			txGas:       params.TxGas + 20000,
			wantGasUsed: params.TxGas + 13579,
		},
		{
			name:                   "error propagation",
			to:                     &common.Address{},
			skipChargeRegistration: true,
			txGas:                  params.TxGas,
			wantGasUsed:            params.TxGas,
			wantVMErr:              errUnknownTx,
		},
	}

	config := params.AllDevChainProtocolChanges
	key, err := crypto.GenerateKey()
	require.NoError(t, err, "crypto.GenerateKey()")
	eoa := crypto.PubkeyToAddress(key.PublicKey)

	header := &types.Header{
		Number:     big.NewInt(0),
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(0),
	}
	signer := types.MakeSigner(config, header.Number, header.Time)

	var txs types.Transactions
	charge := make(map[common.Hash]uint64)
	for i, tt := range tests {
		tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
			// Although nonces aren't strictly necessary, they guarantee a
			// different tx hash for each one.
			Nonce:    uint64(i), //nolint:gosec // Known to not overflow
			To:       tt.to,
			GasPrice: big.NewInt(1),
			Gas:      tt.txGas,
		})
		txs = append(txs, tx)
		if !tt.skipChargeRegistration {
			charge[tx.Hash()] = tt.charge
		}
	}

	vm.RegisterHooks(&preprocessingCharger{
		charge: charge,
	})
	t.Cleanup(vm.TestOnlyClearRegisteredHooks)

	for i, tt := range tests {
		tx := txs[i]

		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Extra gas charge: %d", tt.charge)

			t.Run("ApplyTransaction", func(t *testing.T) {
				_, _, sdb := ethtest.NewEmptyStateDB(t)
				sdb.SetTxContext(tx.Hash(), i)
				sdb.SetBalance(eoa, new(uint256.Int).SetAllOne())
				sdb.SetNonce(eoa, tx.Nonce())

				var gotGasUsed uint64
				gp := core.GasPool(math.MaxUint64)

				receipt, err := core.ApplyTransaction(
					config, ethtest.DummyChainContext(), &common.Address{},
					&gp, sdb, header, tx, &gotGasUsed, vm.Config{},
				)
				require.NoError(t, err, "core.ApplyTransaction(...)")

				wantStatus := types.ReceiptStatusSuccessful
				if tt.wantVMErr != nil {
					wantStatus = types.ReceiptStatusFailed
				}
				assert.Equalf(t, wantStatus, receipt.Status, "%T.Status", receipt)

				assert.Equal(t, tt.wantGasUsed, gotGasUsed, "core.ApplyTransaction(..., &gotGasUsed, ...)")
				assert.Equalf(t, tt.wantGasUsed, receipt.GasUsed, "core.ApplyTransaction(...) -> %T.GasUsed", receipt)
			})

			t.Run("VM_error", func(t *testing.T) {
				sdb, evm := ethtest.NewZeroEVM(t, ethtest.WithChainConfig(config))
				sdb.SetTxContext(tx.Hash(), i)
				sdb.SetBalance(eoa, new(uint256.Int).SetAllOne())
				sdb.SetNonce(eoa, tx.Nonce())

				msg, err := core.TransactionToMessage(tx, signer, header.BaseFee)
				require.NoError(t, err, "core.TransactionToMessage(...)")

				gp := core.GasPool(math.MaxUint64)
				got, err := core.ApplyMessage(evm, msg, &gp)
				require.NoError(t, err, "core.ApplyMessage(...)")
				require.ErrorIsf(t, got.Err, tt.wantVMErr, "%T.Err", got)
			})
		})
	}
}
