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

package core

import (
	"errors"
	"math/big"
	"reflect"
	"runtime"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/params"
)

func TestValidateAuthorizations(t *testing.T) {
	stateDb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	evm := vm.NewEVM(vm.BlockContext{BlockNumber: new(big.Int), Time: new(big.Int)}, vm.TxContext{}, stateDb, params.TestChainConfig, vm.Config{})
	st := &StateTransition{evm: evm, state: stateDb}

	t.Run("Chain ID mismatch", func(t *testing.T) {
		auth := &types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(big.NewInt(2)),
			Nonce:   0,
		}
		_, _, err := st.validateAuthorization(auth)
		assert.Equal(t, ErrAuthorizationWrongChainID, err)
	})

	t.Run("Nonce overflow", func(t *testing.T) {
		auth := &types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(big.NewInt(1)),
			Nonce:   ^uint64(0),
		}
		_, _, err := st.validateAuthorization(auth)
		assert.Equal(t, ErrAuthorizationNonceOverflow, err)
	})

	t.Run("Invalid signature", func(t *testing.T) {
		// gomonkey concurrency issue workaround,
		// see https://github.com/agiledragon/gomonkey/issues/145
		defer runtime.GC()

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc((*types.SetCodeAuthorization).Authority, func(_ *types.SetCodeAuthorization) (common.Address, error) {
			return common.Address{}, errors.New("invalid signature")
		})
		auth := &types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(big.NewInt(1)),
			Nonce:   0,
		}
		_, _, err := st.validateAuthorization(auth)
		assert.ErrorIs(t, err, ErrAuthorizationInvalidSignature)
	})

	t.Run("Destination has code", func(t *testing.T) {
		// gomonkey concurrency issue workaround,
		// see https://github.com/agiledragon/gomonkey/issues/145
		defer runtime.GC()

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc((*types.SetCodeAuthorization).Authority, func(_ *types.SetCodeAuthorization) (common.Address, error) {
			return common.Address{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(st.state),
			"GetCode",
			func(_ interface{}, addr common.Address) []byte {
				return []byte{byte(vm.PUSH1), 0x00, byte(vm.PUSH1), 0x00, byte(vm.RETURN)}
			},
		)
		auth := &types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(big.NewInt(1)),
			Nonce:   0,
		}
		_, _, err := st.validateAuthorization(auth)
		assert.Equal(t, ErrAuthorizationDestinationHasCode, err)
	})

	t.Run("Nonce mismatch", func(t *testing.T) {
		// gomonkey concurrency issue workaround,
		// see https://github.com/agiledragon/gomonkey/issues/145
		defer runtime.GC()

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc((*types.SetCodeAuthorization).Authority, func(_ *types.SetCodeAuthorization) (common.Address, error) {
			return common.Address{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(st.state),
			"GetCode",
			func(_ interface{}, addr common.Address) []byte {
				return nil
			},
		)
		patches.ApplyMethod(reflect.TypeOf(st.state),
			"GetNonce",
			func(_ interface{}, addr common.Address) uint64 {
				return 1
			},
		)
		auth := &types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(big.NewInt(1)),
			Nonce:   0,
		}
		_, _, err := st.validateAuthorization(auth)
		assert.Equal(t, ErrAuthorizationNonceMismatch, err)
	})

	t.Run("Valid authorization", func(t *testing.T) {
		// gomonkey concurrency issue workaround,
		// see https://github.com/agiledragon/gomonkey/issues/145
		defer runtime.GC()

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc((*types.SetCodeAuthorization).Authority, func(_ *types.SetCodeAuthorization) (common.Address, error) {
			return common.Address{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(st.state),
			"GetCode",
			func(_ interface{}, addr common.Address) []byte {
				return nil
			},
		)
		patches.ApplyMethod(reflect.TypeOf(st.state),
			"GetNonce",
			func(_ interface{}, addr common.Address) uint64 {
				return 0
			},
		)
		auth := &types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(big.NewInt(1)),
			Nonce:   0,
		}
		authority, preCode, err := st.validateAuthorization(auth)
		assert.NoError(t, err)
		assert.Equal(t, common.Address{}, authority)
		assert.Equal(t, []byte{}, preCode)

		auth.ChainID = *uint256.MustFromBig(big.NewInt(0))
		authority, preCode, err = st.validateAuthorization(auth)
		assert.NoError(t, err)
		assert.Equal(t, common.Address{}, authority)
		assert.Equal(t, []byte{}, preCode)
	})
}
