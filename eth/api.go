// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

// PublicEthereumApi provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumApi struct {
	e   *Ethereum
	gpo *GasPriceOracle
}

// NewPublicEthereumApi creates a new Etheruem protocol API.
func NewPublicEthereumApi(e *Ethereum) *PublicEthereumApi {
	return &PublicEthereumApi{e, NewGasPriceOracle(e)}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicEthereumApi) GasPrice() *big.Int {
	return s.gpo.SuggestPrice()
}

// GetCompilers returns the collection of available smart contract compilers
func (s *PublicEthereumApi) GetCompilers() ([]string, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc != nil {
		return []string{"Solidity"}, nil
	}

	return nil, nil
}

// CompileSolidity compiles the given solidity source
func (s *PublicEthereumApi) CompileSolidity(source string) (map[string]*compiler.Contract, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc == nil {
		return nil, errors.New("solc (solidity compiler) not found")
	}

	return solc.Compile(source)
}

// Etherbase is the address that mining rewards will be send to
func (s *PublicEthereumApi) Etherbase() (common.Address, error) {
	return s.e.Etherbase()
}

// see Etherbase
func (s *PublicEthereumApi) Coinbase() (common.Address, error) {
	return s.Etherbase()
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *PublicEthereumApi) ProtocolVersion() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.EthVersion())
}

// Hashrate returns the POW hashrate
func (s *PublicEthereumApi) Hashrate() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.Miner().HashRate())
}

// Syncing returns false in case the node is currently not synching with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing an object with 3 properties is
// returned:
// - startingBlock: block number this node started to synchronise from
// - currentBlock: block number this node is currently importing
// - highestBlock: block number of the highest block header this node has received from peers
func (s *PublicEthereumApi) Syncing() (interface{}, error) {
	origin, current, height := s.e.Downloader().Progress()
	if current < height {
		return map[string]interface{}{
			"startingBlock": rpc.NewHexNumber(origin),
			"currentBlock":  rpc.NewHexNumber(current),
			"highestBlock":  rpc.NewHexNumber(height),
		}, nil
	}
	return false, nil
}

// PrivateMinerApi provides private RPC methods to control the miner.
// These methods can be abused by external users and must be considered insecure for use by untrusted users.
type PrivateMinerApi struct {
	e *Ethereum
}

// NewPrivateMinerApi create a new RPC service which controls the miner of this node.
func NewPrivateMinerApi(e *Ethereum) *PrivateMinerApi {
	return &PrivateMinerApi{e: e}
}

// Start the miner with the given number of threads
func (s *PrivateMinerApi) Start(threads rpc.HexNumber) (bool, error) {
	s.e.StartAutoDAG()
	err := s.e.StartMining(threads.Int(), "")
	if err == nil {
		return true, nil
	}
	return false, err
}

// Stop the miner
func (s *PrivateMinerApi) Stop() bool {
	s.e.StopMining()
	return true
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (s *PrivateMinerApi) SetExtra(extra string) (bool, error) {
	if err := s.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (s *PrivateMinerApi) SetGasPrice(gasPrice rpc.Number) bool {
	s.e.Miner().SetGasPrice(gasPrice.BigInt())
	return true
}

// SetEtherbase sets the etherbase of the miner
func (s *PrivateMinerApi) SetEtherbase(etherbase common.Address) bool {
	s.e.SetEtherbase(etherbase)
	return true
}

// StartAutoDAG starts auto DAG generation. This will prevent the DAG generating on epoch change
// which will cause the node to stop mining during the generation process.
func (s *PrivateMinerApi) StartAutoDAG() bool {
	s.e.StartAutoDAG()
	return true
}

// StopAutoDAG stops auto DAG generation
func (s *PrivateMinerApi) StopAutoDAG() bool {
	s.e.StopAutoDAG()
	return true
}

// MakeDAG creates the new DAG for the given block number
func (s *PrivateMinerApi) MakeDAG(blockNr rpc.BlockNumber) (bool, error) {
	if err := ethash.MakeDAG(uint64(blockNr.Int64()), ""); err != nil {
		return false, err
	}
	return true, nil
}

// PublicTxPoolApi offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolApi struct {
	e *Ethereum
}

// NewPublicTxPoolApi creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolApi(e *Ethereum) *PublicTxPoolApi {
	return &PublicTxPoolApi{e}
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolApi) Status() map[string]*rpc.HexNumber {
	pending, queue := s.e.TxPool().Stats()
	return map[string]*rpc.HexNumber{
		"pending": rpc.NewHexNumber(pending),
		"queued":  rpc.NewHexNumber(queue),
	}
}

// PublicAccountApi provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountApi struct {
	am *accounts.Manager
}

// NewPublicAccountApi creates a new PublicAccountApi.
func NewPublicAccountApi(am *accounts.Manager) *PublicAccountApi {
	return &PublicAccountApi{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountApi) Accounts() ([]accounts.Account, error) {
	return s.am.Accounts()
}

// PrivateAccountApi provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts.
type PrivateAccountApi struct {
	am *accounts.Manager
}

// NewPrivateAccountApi create a new PrivateAccountApi.
func NewPrivateAccountApi(am *accounts.Manager) *PrivateAccountApi {
	return &PrivateAccountApi{am}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountApi) ListAccounts() ([]common.Address, error) {
	accounts, err := s.am.Accounts()
	if err != nil {
		return nil, err
	}

	addresses := make([]common.Address, len(accounts))
	for i, acc := range accounts {
		addresses[i] = acc.Address
	}
	return addresses, nil
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountApi) NewAccount(password string) (common.Address, error) {
	acc, err := s.am.NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// UnlockAccount will unlock the account associated with the given address with the given password for duration seconds.
// It returns an indication if the action was successful.
func (s *PrivateAccountApi) UnlockAccount(addr common.Address, password string, duration int) bool {
	if err := s.am.TimedUnlock(addr, password, time.Duration(duration)*time.Second); err != nil {
		glog.V(logger.Info).Infof("%v\n", err)
		return false
	}
	return true
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountApi) LockAccount(addr common.Address) bool {
	return s.am.Lock(addr) == nil
}
