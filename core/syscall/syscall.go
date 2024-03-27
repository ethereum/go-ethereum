// Copyright 2024 The go-ethereum Authors
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

package syscall

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var sysTxContext = vm.TxContext{
	Origin:   params.SystemAddress,
	GasPrice: common.Big0,
}

func newBlockContext(header *types.Header) vm.BlockContext {
	return vm.BlockContext{
		CanTransfer: func(db vm.StateDB, addr common.Address, amount *uint256.Int) bool { return false },
		Transfer:    func(vm.StateDB, common.Address, common.Address, *uint256.Int) {},
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: header.Number,
		Time:        header.Time,
		Difficulty:  common.Big0,
		BaseFee:     common.Big0,
		BlobBaseFee: common.Big0,
		GasLimit:    30_000_000,
		Random:      &common.Hash{},
	}
}

// RunPreBlockHooks executes all relevant pre-block operations. It will update statedb.
func RunPreBlockHooks(header *types.Header, statedb *state.StateDB, config *params.ChainConfig) {
	ProcessBeaconBlockRoot(header, statedb, config)
}

// ProcessBeaconBlockRoot applies the EIP-4788 system call to the beacon block root
// contract. This method is exported to be used in tests.
func ProcessBeaconBlockRoot(header *types.Header, statedb *state.StateDB, config *params.ChainConfig) {
	if header.ParentBeaconRoot == nil {
		return
	}
	// If EIP-4788 is enabled, we need to invoke the beaconroot storage contract with
	// the new root
	statedb.AddAddressToAccessList(params.BeaconRootsAddress)
	evm := vm.NewEVM(newBlockContext(header), sysTxContext, statedb, config, vm.Config{})
	evm.Call(vm.AccountRef(params.SystemAddress), params.BeaconRootsAddress, header.ParentBeaconRoot.Bytes(), 30_000_000, common.U2560)
	statedb.Finalise(true)
}
