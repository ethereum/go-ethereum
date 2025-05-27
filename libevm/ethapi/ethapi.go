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

// Package ethapi exposes the internal ethapi package.
package ethapi

import "github.com/ava-labs/libevm/internal/ethapi"

// Type aliases required by constructors.
type (
	Backend    = ethapi.Backend
	AddrLocker = ethapi.AddrLocker
)

type (
	// BlockChainAPI exposes RPC methods for querying chain data.
	BlockChainAPI = ethapi.BlockChainAPI
	// TransactionAPI exposes RPC methods for querying and creating
	// transactions.
	TransactionAPI = ethapi.TransactionAPI
)

// NewBlockChainAPI is identical to [ethapi.NewBlockChainAPI].
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return ethapi.NewBlockChainAPI(b)
}

// NewTransactionAPI is identical to [ethapi.NewTransactionAPI].
func NewTransactionAPI(b Backend, nonceLock *AddrLocker) *TransactionAPI {
	return ethapi.NewTransactionAPI(b, nonceLock)
}
