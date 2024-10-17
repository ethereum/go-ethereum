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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/libevm/hookstest"
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
