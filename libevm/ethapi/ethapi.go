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
	// EthereumAPI provides an API to access Ethereum related information.
	EthereumAPI = ethapi.EthereumAPI
	// BlockChainAPI provides an API to access Ethereum blockchain data.
	BlockChainAPI = ethapi.BlockChainAPI
	// TransactionAPI exposes methods for reading and creating transaction data.
	TransactionAPI = ethapi.TransactionAPI
	// TxPoolAPI offers and API for the transaction pool. It only operates on
	// data that is non-confidential.
	TxPoolAPI = ethapi.TxPoolAPI
	// DebugAPI is the collection of Ethereum APIs exposed over the debugging
	// namespace.
	DebugAPI = ethapi.DebugAPI
)

// Type aliases for types used as arguments or responses to the APIs.
type (
	RPCTransaction = ethapi.RPCTransaction
)

// NewEthereumAPI is identical to [ethapi.NewEthereumAPI].
func NewEthereumAPI(b Backend) *EthereumAPI {
	return ethapi.NewEthereumAPI(b)
}

// NewBlockChainAPI is identical to [ethapi.NewBlockChainAPI].
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return ethapi.NewBlockChainAPI(b)
}

// NewTransactionAPI is identical to [ethapi.NewTransactionAPI].
func NewTransactionAPI(b Backend, nonceLock *AddrLocker) *TransactionAPI {
	return ethapi.NewTransactionAPI(b, nonceLock)
}

// NewTxPoolAPI is identical to [ethapi.NewTxPoolAPI].
func NewTxPoolAPI(b Backend) *TxPoolAPI {
	return ethapi.NewTxPoolAPI(b)
}

// NewDebugAPI is identical to [ethapi.NewDebugAPI].
func NewDebugAPI(b Backend) *DebugAPI {
	return ethapi.NewDebugAPI(b)
}
