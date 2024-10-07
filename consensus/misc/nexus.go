// Copyright 2016 The go-ethereum Authors
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

package misc

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
    "github.com/ethereum/go-ethereum/core/tracing"
    "github.com/holiman/uint256"
)


// ApplyNexusHardFork modifies the state database according to the Nexus hard-fork
// rules, transferring SYS balance from previous VaultManager to new one
func ApplyNexusHardFork(statedb *state.StateDB) {
    // Create the new contract account if it doesn't already exist
    if !statedb.Exist(params.VaultManager) {
        statedb.CreateAccount(params.VaultManager)
    }

    // Transfer the balance from the old contract to the new contract
    oldBalance := statedb.GetBalance(params.VaultManagerOld)
    statedb.AddBalance(params.VaultManager, oldBalance, tracing.BalanceIncreaseVaultManagerContract)
    statedb.SetBalance(params.VaultManagerOld, new(uint256.Int), tracing.BalanceDecreaseVaultManagerAccount)// Reset the old contract's balance

}