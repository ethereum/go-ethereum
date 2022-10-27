// Copyright 2015 The go-ethereum Authors
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

package ethapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/common/math"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	contractValidator "github.com/XinFinOrg/XDPoSChain/contracts/validator/contract"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	defaultGasPrice = 50 * params.Shannon
	// statuses of candidates
	statusMasternode = "MASTERNODE"
	statusSlashed    = "SLASHED"
	statusProposed   = "PROPOSED"
	fieldStatus      = "status"
	fieldCapacity    = "capacity"
	fieldCandidates  = "candidates"
	fieldSuccess     = "success"
	fieldEpoch       = "epoch"
)

var errEmptyHeader = errors.New("empty header")

// PublicEthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumAPI struct {
	b Backend
}

// NewPublicEthereumAPI creates a new Ethereum protocol API.
func NewPublicEthereumAPI(b Backend) *PublicEthereumAPI {
	return &PublicEthereumAPI{b}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicEthereumAPI) GasPrice(ctx context.Context) (*big.Int, error) {
	return s.b.SuggestPrice(ctx)
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *PublicEthereumAPI) ProtocolVersion() hexutil.Uint {
	return hexutil.Uint(s.b.ProtocolVersion())
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *PublicEthereumAPI) Syncing() (interface{}, error) {
	progress := s.b.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock": hexutil.Uint64(progress.StartingBlock),
		"currentBlock":  hexutil.Uint64(progress.CurrentBlock),
		"highestBlock":  hexutil.Uint64(progress.HighestBlock),
		"pulledStates":  hexutil.Uint64(progress.PulledStates),
		"knownStates":   hexutil.Uint64(progress.KnownStates),
	}, nil
}

// PublicTxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolAPI struct {
	b Backend
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolAPI(b Backend) *PublicTxPoolAPI {
	return &PublicTxPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction),
		"queued":  make(map[string]map[string]*RPCTransaction),
	}
	pending, queue := s.b.TxPoolContent()

	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := s.b.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *PublicTxPoolAPI) Inspect() map[string]map[string]map[string]string {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	pending, queue := s.b.TxPoolContent()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// PublicAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountAPI struct {
	am *accounts.Manager
}

// NewPublicAccountAPI creates a new PublicAccountAPI.
func NewPublicAccountAPI(am *accounts.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountAPI) Accounts() []common.Address {
	addresses := make([]common.Address, 0) // return [] instead of nil if empty
	for _, wallet := range s.am.Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}
	return addresses
}

// PrivateAccountAPI provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts. Some methods accept
// passwords and are therefore considered private by default.
type PrivateAccountAPI struct {
	am        *accounts.Manager
	nonceLock *AddrLocker
	b         Backend
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(b Backend, nonceLock *AddrLocker) *PrivateAccountAPI {
	return &PrivateAccountAPI{
		am:        b.AccountManager(),
		nonceLock: nonceLock,
		b:         b,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() []common.Address {
	addresses := make([]common.Address, 0) // return [] instead of nil if empty
	for _, wallet := range s.am.Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}
	return addresses
}

// rawWallet is a JSON representation of an accounts.Wallet interface, with its
// data contents extracted into plain fields.
type rawWallet struct {
	URL      string             `json:"url"`
	Status   string             `json:"status"`
	Failure  string             `json:"failure,omitempty"`
	Accounts []accounts.Account `json:"accounts,omitempty"`
}

// ListWallets will return a list of wallets this node manages.
func (s *PrivateAccountAPI) ListWallets() []rawWallet {
	wallets := make([]rawWallet, 0) // return [] instead of nil if empty
	for _, wallet := range s.am.Wallets() {
		status, failure := wallet.Status()

		raw := rawWallet{
			URL:      wallet.URL().String(),
			Status:   status,
			Accounts: wallet.Accounts(),
		}
		if failure != nil {
			raw.Failure = failure.Error()
		}
		wallets = append(wallets, raw)
	}
	return wallets
}

// OpenWallet initiates a hardware wallet opening procedure, establishing a USB
// connection and attempting to authenticate via the provided passphrase. Note,
// the method may return an extra challenge requiring a second open (e.g. the
// Trezor PIN matrix challenge).
func (s *PrivateAccountAPI) OpenWallet(url string, passphrase *string) error {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return err
	}
	pass := ""
	if passphrase != nil {
		pass = *passphrase
	}
	return wallet.Open(pass)
}

// DeriveAccount requests a HD wallet to derive a new account, optionally pinning
// it for later reuse.
func (s *PrivateAccountAPI) DeriveAccount(url string, path string, pin *bool) (accounts.Account, error) {
	wallet, err := s.am.Wallet(url)
	if err != nil {
		return accounts.Account{}, err
	}
	derivPath, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return accounts.Account{}, err
	}
	if pin == nil {
		pin = new(bool)
	}
	return wallet.Derive(derivPath, *pin)
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (common.Address, error) {
	acc, err := fetchKeystore(s.am).NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// fetchKeystore retrives the encrypted keystore from the account manager.
func fetchKeystore(am *accounts.Manager) *keystore.KeyStore {
	return am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (s *PrivateAccountAPI) ImportRawKey(privkey string, password string) (common.Address, error) {
	key, err := crypto.HexToECDSA(privkey)
	if err != nil {
		return common.Address{}, err
	}
	acc, err := fetchKeystore(s.am).ImportECDSA(key, password)
	return acc.Address, err
}

// UnlockAccount will unlock the account associated with the given address with
// the given password for duration seconds. If duration is nil it will use a
// default of 300 seconds. It returns an indication if the account was unlocked.
func (s *PrivateAccountAPI) UnlockAccount(addr common.Address, password string, duration *uint64) (bool, error) {
	const max = uint64(time.Duration(math.MaxInt64) / time.Second)
	var d time.Duration
	if duration == nil {
		d = 300 * time.Second
	} else if *duration > max {
		return false, errors.New("unlock duration too large")
	} else {
		d = time.Duration(*duration) * time.Second
	}
	err := fetchKeystore(s.am).TimedUnlock(accounts.Account{Address: addr}, password, d)
	return err == nil, err
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr common.Address) bool {
	return fetchKeystore(s.am).Lock(addr) == nil
}

// signTransactions sets defaults and signs the given transaction
// NOTE: the caller needs to ensure that the nonceLock is held, if applicable,
// and release it after the transaction has been submitted to the tx pool
func (s *PrivateAccountAPI) signTransaction(ctx context.Context, args SendTxArgs, passwd string) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}
	wallet, err := s.am.Find(account)
	if err != nil {
		return nil, err
	}
	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	var chainID *big.Int
	if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number()) {
		chainID = config.ChainId
	}
	return wallet.SignTxWithPassphrase(account, passwd, tx, chainID)
}

// SendTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails.
func (s *PrivateAccountAPI) SendTransaction(ctx context.Context, args SendTxArgs, passwd string) (common.Hash, error) {
	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}
	signed, err := s.signTransaction(ctx, args, passwd)
	if err != nil {
		return common.Hash{}, err
	}
	return submitTransaction(ctx, s.b, signed)
}

// SignTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails. The transaction is returned in RLP-form, not broadcast
// to other nodes
func (s *PrivateAccountAPI) SignTransaction(ctx context.Context, args SendTxArgs, passwd string) (*SignTransactionResult, error) {
	// No need to obtain the noncelock mutex, since we won't be sending this
	// tx into the transaction pool, but right back to the user
	if args.Gas == nil {
		return nil, fmt.Errorf("gas not specified")
	}
	if args.GasPrice == nil {
		return nil, fmt.Errorf("gasPrice not specified")
	}
	if args.Nonce == nil {
		return nil, fmt.Errorf("nonce not specified")
	}
	signed, err := s.signTransaction(ctx, args, passwd)
	if err != nil {
		return nil, err
	}
	data, err := rlp.EncodeToBytes(signed)
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, signed}, nil
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

// Sign calculates an Ethereum ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message))
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/XinFinOrg/XDPoSChain/wiki/Management-APIs#personal_sign
func (s *PrivateAccountAPI) Sign(ctx context.Context, data hexutil.Bytes, addr common.Address, passwd string) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Assemble sign the data with the wallet
	signature, err := wallet.SignHashWithPassphrase(account, passwd, signHash(data))
	if err != nil {
		return nil, err
	}
	signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return signature, nil
}

// EcRecover returns the address for the account that was used to create the signature.
// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
// the address of:
// hash = keccak256("\x19Ethereum Signed Message:\n"${message length}${message})
// addr = ecrecover(hash, signature)
//
// Note, the signature must conform to the secp256k1 curve R, S and V values, where
// the V value must be be 27 or 28 for legacy reasons.
//
// https://github.com/XinFinOrg/XDPoSChain/wiki/Management-APIs#personal_ecRecover
func (s *PrivateAccountAPI) EcRecover(ctx context.Context, data, sig hexutil.Bytes) (common.Address, error) {
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}
	if sig[64] != 27 && sig[64] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1

	rpk, err := crypto.Ecrecover(signHash(data), sig)
	if err != nil {
		return common.Address{}, err
	}
	pubKey := crypto.ToECDSAPub(rpk)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr, nil
}

// SignAndSendTransaction was renamed to SendTransaction. This method is deprecated
// and will be removed in the future. It primary goal is to give clients time to update.
func (s *PrivateAccountAPI) SignAndSendTransaction(ctx context.Context, args SendTxArgs, passwd string) (common.Hash, error) {
	return s.SendTransaction(ctx, args, passwd)
}

// PublicBlockChainAPI provides an API to access the Ethereum blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainAPI struct {
	b Backend
}

// NewPublicBlockChainAPI creates a new Ethereum blockchain API.
func NewPublicBlockChainAPI(b Backend) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{b}
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) BlockNumber() *big.Int {
	header, _ := s.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return header.Number
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) GetRewardByHash(hash common.Hash) map[string]map[string]map[string]*big.Int {
	return s.b.GetRewardByHash(hash)
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (s *PublicBlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*big.Int, error) {
	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	b := state.GetBalance(address)
	return b, state.Error()
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(ctx context.Context, blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := s.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		response, err := s.rpcOutputBlock(block, true, fullTx, ctx)
		if err == nil && blockNr == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(ctx context.Context, blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := s.b.GetBlock(ctx, blockHash)
	if block != nil {
		return s.rpcOutputBlock(block, true, fullTx, ctx)
	}
	return nil, err
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash(), "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return s.rpcOutputBlock(block, false, false, ctx)
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
// DEPRECATED SINCE 1.0
func (s *PublicBlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := s.b.GetBlock(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", block.Number(), "hash", blockHash, "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return s.rpcOutputBlock(block, false, false, ctx)
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
// DEPRECATED SINCE 1.0
func (s *PublicBlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
// DEPRECATED SINCE 1.0
func (s *PublicBlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *PublicBlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, key string, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	res := state.GetState(address, common.HexToHash(key))
	return res[:], state.Error()
}

func (s *PublicBlockChainAPI) GetBlockSignersByHash(ctx context.Context, blockHash common.Hash) ([]common.Address, error) {
	block, err := s.b.GetBlock(ctx, blockHash)
	if err != nil || block == nil {
		return []common.Address{}, err
	}
	masternodes, err := s.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return []common.Address{}, err
	}
	return s.rpcOutputBlockSigners(block, ctx, masternodes)
}

func (s *PublicBlockChainAPI) GetBlockSignersByNumber(ctx context.Context, blockNumber rpc.BlockNumber) ([]common.Address, error) {
	block, err := s.b.BlockByNumber(ctx, blockNumber)
	if err != nil || block == nil {
		return []common.Address{}, err
	}
	masternodes, err := s.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return []common.Address{}, err
	}
	return s.rpcOutputBlockSigners(block, ctx, masternodes)
}

func (s *PublicBlockChainAPI) GetBlockFinalityByHash(ctx context.Context, blockHash common.Hash) (uint, error) {
	block, err := s.b.GetBlock(ctx, blockHash)
	if err != nil || block == nil {
		return uint(0), err
	}
	masternodes, err := s.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return uint(0), err
	}
	return s.findFinalityOfBlock(ctx, block, masternodes)
}

func (s *PublicBlockChainAPI) GetBlockFinalityByNumber(ctx context.Context, blockNumber rpc.BlockNumber) (uint, error) {
	block, err := s.b.BlockByNumber(ctx, blockNumber)
	if err != nil || block == nil {
		return uint(0), err
	}
	masternodes, err := s.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return uint(0), err
	}
	return s.findFinalityOfBlock(ctx, block, masternodes)
}

// GetMasternodes returns masternodes set at the starting block of epoch of the given block
func (s *PublicBlockChainAPI) GetMasternodes(ctx context.Context, b *types.Block) ([]common.Address, error) {
	var masternodes []common.Address
	if b.Number().Int64() >= 0 {
		curBlockNumber := b.Number().Uint64()
		prevBlockNumber := curBlockNumber + (common.MergeSignRange - (curBlockNumber % common.MergeSignRange))
		latestBlockNumber := s.b.CurrentBlock().Number().Uint64()
		if prevBlockNumber >= latestBlockNumber || !s.b.ChainConfig().IsTIP2019(b.Number()) {
			prevBlockNumber = curBlockNumber
		}
		if engine, ok := s.b.GetEngine().(*XDPoS.XDPoS); ok {
			// Get block epoc latest.
			lastCheckpointNumber := prevBlockNumber - (prevBlockNumber % s.b.ChainConfig().XDPoS.Epoch)
			prevCheckpointBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(lastCheckpointNumber))
			if prevCheckpointBlock != nil {
				masternodes = engine.GetMasternodesFromCheckpointHeader(prevCheckpointBlock.Header(), curBlockNumber, s.b.ChainConfig().XDPoS.Epoch)
			}
		} else {
			log.Error("Undefined XDPoS consensus engine")
		}
	}
	return masternodes, nil
}

// GetCandidateStatus returns status of the given candidate at a specified epochNumber
func (s *PublicBlockChainAPI) GetCandidateStatus(ctx context.Context, coinbaseAddress common.Address, epoch rpc.EpochNumber) (map[string]interface{}, error) {
	var (
		block                    *types.Block
		header                   *types.Header
		checkpointNumber         rpc.BlockNumber
		epochNumber              rpc.EpochNumber // if epoch == "latest", print the latest epoch number to epochNumber
		masternodes, penaltyList []common.Address
		candidates               []utils.Masternode
		penalties                []byte
		err                      error
	)

	result := map[string]interface{}{
		fieldStatus:   "",
		fieldCapacity: 0,
		fieldSuccess:  true,
	}

	epochConfig := s.b.ChainConfig().XDPoS.Epoch

	// checkpoint block
	checkpointNumber, epochNumber = s.GetPreviousCheckpointFromEpoch(ctx, epoch)
	result[fieldEpoch] = epochNumber.Int64()

	block, err = s.b.BlockByNumber(ctx, checkpointNumber)
	if err != nil || block == nil { // || checkpointNumber == 0 {
		result[fieldSuccess] = false
		return result, err
	}

	header = block.Header()
	if header == nil {
		log.Error("Empty header at checkpoint ", "num", checkpointNumber)
		return result, errEmptyHeader
	}

	// list of candidates (masternode, slash, propose) at block checkpoint
	if epoch == rpc.LatestEpochNumber {
		candidates, err = s.getCandidatesFromSmartContract()
	} else {
		statedb, _, err := s.b.StateAndHeaderByNumber(ctx, checkpointNumber)
		if err != nil {
			result[fieldSuccess] = false
			return result, err
		}
		candidatesAddresses := state.GetCandidates(statedb)
		for _, address := range candidatesAddresses {
			v := state.GetCandidateCap(statedb, address)
			if address.String() != "0x0000000000000000000000000000000000000000" {
				candidates = append(candidates, utils.Masternode{Address: address, Stake: v})
			}
		}
	}
	if err != nil || len(candidates) == 0 {
		log.Debug("Candidates list cannot be found", "len(candidates)", len(candidates), "err", err)
		result[fieldSuccess] = false
		return result, err
	}
	var maxMasternodes int
	if s.b.ChainConfig().IsTIPIncreaseMasternodes(block.Number()) {
		maxMasternodes = common.MaxMasternodesV2
	} else {
		maxMasternodes = common.MaxMasternodes
	}

	isTopCandidate := false
	// check penalties from checkpoint headers and modify status of a node to SLASHED if it's in top 150 candidates
	// if it's SLASHED but it's out of top 150, the status should be still PROPOSED
	for i := 0; i < len(candidates); i++ {
		if coinbaseAddress == candidates[i].Address {
			if i < maxMasternodes {
				isTopCandidate = true
			}
			result[fieldStatus] = statusProposed
			result[fieldCapacity] = candidates[i].Stake
			break
		}
	}
	if !isTopCandidate {
		return result, nil
	}

	// Second, Find candidates that have masternode status
	if engine, ok := s.b.GetEngine().(*XDPoS.XDPoS); ok {
		masternodes = engine.GetMasternodesFromCheckpointHeader(header, block.Number().Uint64(), s.b.ChainConfig().XDPoS.Epoch)
		if len(masternodes) == 0 {
			log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes), "blockNum", header.Number.Uint64())
			result[fieldSuccess] = false
			return result, err
		}
	} else {
		log.Error("Undefined XDPoS consensus engine")
	}
	// Set masternode status
	for _, masternode := range masternodes {
		if coinbaseAddress == masternode {
			result[fieldStatus] = statusMasternode
			return result, nil
		}
	}

	// Third, Get penalties list
	penalties = append(penalties, header.Penalties...)
	// check last 5 epochs to find penalize masternodes
	for i := 1; i <= common.LimitPenaltyEpoch; i++ {
		if header.Number.Uint64() < epochConfig*uint64(i) {
			break
		}
		blockNum := header.Number.Uint64() - epochConfig*uint64(i)
		checkpointHeader, err := s.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNum))
		if checkpointHeader == nil || err != nil {
			log.Error("Failed to get header by number", "num", blockNum, "err", err)
			continue
		}
		penalties = append(penalties, checkpointHeader.Penalties...)
	}
	penaltyList = common.ExtractAddressFromBytes(penalties)

	// map slashing status
	for _, pen := range penaltyList {
		if coinbaseAddress == pen {
			result[fieldStatus] = statusSlashed
			return result, nil
		}
	}
	return result, nil
}

// GetCandidates returns status of all candidates at a specified epochNumber
func (s *PublicBlockChainAPI) GetCandidates(ctx context.Context, epoch rpc.EpochNumber) (map[string]interface{}, error) {
	var (
		block            *types.Block
		header           *types.Header
		checkpointNumber rpc.BlockNumber
		epochNumber      rpc.EpochNumber
		masternodes      []common.Address
		penaltyList      []common.Address
		candidates       []utils.Masternode
		penalties        []byte
		err              error
	)
	result := map[string]interface{}{
		fieldSuccess: true,
	}
	epochConfig := s.b.ChainConfig().XDPoS.Epoch
	candidatesStatusMap := map[string]map[string]interface{}{}

	checkpointNumber, epochNumber = s.GetPreviousCheckpointFromEpoch(ctx, epoch)
	result[fieldEpoch] = epochNumber.Int64()

	block, err = s.b.BlockByNumber(ctx, checkpointNumber)
	if err != nil || block == nil { // || checkpointNumber == 0 {
		result[fieldSuccess] = false
		return result, err
	}

	header = block.Header()

	if header == nil {
		log.Error("Empty header at checkpoint", "num", checkpointNumber)
		return result, errEmptyHeader
	}
	// list of candidates (masternode, slash, propose) at block checkpoint
	if epoch == rpc.LatestEpochNumber {
		candidates, err = s.getCandidatesFromSmartContract()
	} else {
		statedb, _, err := s.b.StateAndHeaderByNumber(ctx, checkpointNumber)
		if err != nil {
			result[fieldSuccess] = false
			return result, err
		}
		candidatesAddresses := state.GetCandidates(statedb)
		for _, address := range candidatesAddresses {
			v := state.GetCandidateCap(statedb, address)
			if address.String() != "0x0000000000000000000000000000000000000000" {
				candidates = append(candidates, utils.Masternode{Address: address, Stake: v})
			}
		}
	}

	if err != nil || len(candidates) == 0 {
		log.Debug("Candidates list cannot be found", "len(candidates)", len(candidates), "err", err)
		result[fieldSuccess] = false
		return result, err
	}
	// First, set all candidate to propose
	for _, candidate := range candidates {
		candidatesStatusMap[candidate.Address.String()] = map[string]interface{}{
			fieldStatus:   statusProposed,
			fieldCapacity: candidate.Stake,
		}
	}

	// Second, Find candidates that have masternode status
	if engine, ok := s.b.GetEngine().(*XDPoS.XDPoS); ok {
		masternodes = engine.GetMasternodesFromCheckpointHeader(header, block.Number().Uint64(), s.b.ChainConfig().XDPoS.Epoch)
		if len(masternodes) == 0 {
			log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes), "blockNum", header.Number.Uint64())
			result[fieldSuccess] = false
			return result, err
		}
	} else {
		log.Error("Undefined XDPoS consensus engine")
	}
	// Set masternode status
	for _, masternode := range masternodes {
		if candidatesStatusMap[masternode.String()] != nil {
			candidatesStatusMap[masternode.String()][fieldStatus] = statusMasternode
		}
	}

	// Third, Get penalties list
	penalties = append(penalties, header.Penalties...)
	// check last 5 epochs to find penalize masternodes
	for i := 1; i <= common.LimitPenaltyEpoch; i++ {
		if header.Number.Uint64() < epochConfig*uint64(i) {
			break
		}
		blockNum := header.Number.Uint64() - epochConfig*uint64(i)
		checkpointHeader, err := s.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNum))
		if checkpointHeader == nil || err != nil {
			log.Error("Failed to get header by number", "num", blockNum, "err", err)
			continue
		}
		penalties = append(penalties, checkpointHeader.Penalties...)
	}
	// map slashing status
	if len(penalties) == 0 {
		result[fieldCandidates] = candidatesStatusMap
		return result, nil
	}
	penaltyList = common.ExtractAddressFromBytes(penalties)

	var topCandidates []utils.Masternode
	if len(candidates) > common.MaxMasternodes {
		topCandidates = candidates[:common.MaxMasternodes]
	} else {
		topCandidates = candidates
	}
	// check penalties from checkpoint headers and modify status of a node to SLASHED if it's in top 150 candidates
	// if it's SLASHED but it's out of top 150, the status should be still PROPOSED
	for _, pen := range penaltyList {
		for _, candidate := range topCandidates {
			if candidate.Address == pen && candidatesStatusMap[pen.String()] != nil {
				candidatesStatusMap[pen.String()][fieldStatus] = statusSlashed
			}
			penalties = append(penalties, block.Penalties()...)
		}
	}

	// update result
	result[fieldCandidates] = candidatesStatusMap
	return result, nil
}

// GetPreviousCheckpointFromEpoch returns header of the previous checkpoint
func (s *PublicBlockChainAPI) GetPreviousCheckpointFromEpoch(ctx context.Context, epochNum rpc.EpochNumber) (rpc.BlockNumber, rpc.EpochNumber) {
	var checkpointNumber uint64
	epoch := s.b.ChainConfig().XDPoS.Epoch

	if epochNum == rpc.LatestEpochNumber {
		blockNumer := s.b.CurrentBlock().Number().Uint64()
		diff := blockNumer % epoch
		// checkpoint number
		checkpointNumber = blockNumer - diff
		epochNum = rpc.EpochNumber(checkpointNumber / epoch)
		if diff > 0 {
			epochNum += 1
		}
	} else if epochNum < 2 {
		checkpointNumber = 0
	} else {
		checkpointNumber = epoch * (uint64(epochNum) - 1)
	}
	return rpc.BlockNumber(checkpointNumber), epochNum
}

// getCandidatesFromSmartContract returns all candidates with their capacities at the current time
func (s *PublicBlockChainAPI) getCandidatesFromSmartContract() ([]utils.Masternode, error) {
	client, err := s.b.GetIPCClient()
	if err != nil {
		return []utils.Masternode{}, err
	}

	addr := common.HexToAddress(common.MasternodeVotingSMC)
	validator, err := contractValidator.NewXDCValidator(addr, client)
	if err != nil {
		return []utils.Masternode{}, err
	}

	opts := new(bind.CallOpts)
	candidates, err := validator.GetCandidates(opts)
	if err != nil {
		return []utils.Masternode{}, err
	}

	var candidatesWithStakeInfo []utils.Masternode

	for _, candidate := range candidates {
		v, err := validator.GetCandidateCap(opts, candidate)
		if err != nil {
			return []utils.Masternode{}, err
		}
		if candidate.String() != "0x0000000000000000000000000000000000000000" {
			candidatesWithStakeInfo = append(candidatesWithStakeInfo, utils.Masternode{Address: candidate, Stake: v})
		}

		if len(candidatesWithStakeInfo) > 0 {
			sort.Slice(candidatesWithStakeInfo, func(i, j int) bool {
				return candidatesWithStakeInfo[i].Stake.Cmp(candidatesWithStakeInfo[j].Stake) >= 0
			})
		}
	}
	return candidatesWithStakeInfo, nil
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      hexutil.Uint64  `json:"gas"`
	GasPrice hexutil.Big     `json:"gasPrice"`
	Value    hexutil.Big     `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

func (s *PublicBlockChainAPI) doCall(ctx context.Context, args CallArgs, blockNr rpc.BlockNumber, vmCfg vm.Config, timeout time.Duration) ([]byte, uint64, bool, error, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	statedb, header, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if statedb == nil || err != nil {
		return nil, 0, false, err, nil
	}
	// Set sender address or use a default if none specified
	addr := args.From
	if addr == (common.Address{}) {
		if wallets := s.b.AccountManager().Wallets(); len(wallets) > 0 {
			if accounts := wallets[0].Accounts(); len(accounts) > 0 {
				addr = accounts[0].Address
			}
		}
	}
	// Set default gas & gas price if none were set
	gas, gasPrice := uint64(args.Gas), args.GasPrice.ToInt()
	if gas == 0 {
		gas = math.MaxUint64 / 2
	}
	if gasPrice.Sign() == 0 {
		gasPrice = new(big.Int).SetUint64(defaultGasPrice)
	}
	balanceTokenFee := big.NewInt(0).SetUint64(gas)
	balanceTokenFee = balanceTokenFee.Mul(balanceTokenFee, gasPrice)
	// Create new call message
	msg := types.NewMessage(addr, args.To, 0, args.Value.ToInt(), gas, gasPrice, args.Data, false, balanceTokenFee)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	block, err := s.b.BlockByNumber(ctx, blockNr)
	if err != nil {
		return nil, 0, false, err, nil
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, 0, false, err, nil
	}
	XDCxState, err := s.b.XDCxService().GetTradingState(block, author)
	if err != nil {
		return nil, 0, false, err, nil
	}
	// Get a new instance of the EVM.
	evm, vmError, err := s.b.GetEVM(ctx, msg, statedb, XDCxState, header, vmCfg)
	if err != nil {
		return nil, 0, false, err, nil
	}
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	owner := common.Address{}
	res, gas, failed, err, vmErr := core.ApplyMessage(evm, msg, gp, owner)
	if err := vmError(); err != nil {
		return nil, 0, false, err, nil
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, 0, false, fmt.Errorf("execution aborted (timeout = %v)", timeout), nil
	}
	if err != nil {
		return res, 0, false, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas()), nil
	}
	return res, gas, failed, err, vmErr
}

func newRevertError(res []byte) *revertError {
	reason, errUnpack := abi.UnpackRevert(res)
	err := errors.New("execution reverted")
	if errUnpack == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return &revertError{
		error:  err,
		reason: hexutil.Encode(res),
	}
}

// revertError is an API error that encompasses an EVM revertal with JSON error
// code and a binary data blob.
type revertError struct {
	error
	reason string // revert reason hex encoded
}

// ErrorCode returns the JSON error code for a revertal.
// See: https://github.com/ethereum/wiki/wiki/JSON-RPC-Error-Codes-Improvement-Proposal
func (e *revertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *revertError) ErrorData() interface{} {
	return e.reason
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(ctx context.Context, args CallArgs, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	result, _, failed, err, vmErr := s.doCall(ctx, args, blockNr, vm.Config{}, 5*time.Second)
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if failed && len(result) > 0 {
		return nil, newRevertError(result)
	}

	return (hexutil.Bytes)(result), vmErr
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (s *PublicBlockChainAPI) EstimateGas(ctx context.Context, args CallArgs) (hexutil.Uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if uint64(args.Gas) >= params.TxGas {
		hi = uint64(args.Gas)
	} else {
		// Retrieve the current pending block to act as the gas ceiling
		block, err := s.b.BlockByNumber(ctx, rpc.LatestBlockNumber)
		if err != nil {
			return 0, err
		}
		hi = block.GasLimit()
	}
	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, []byte, error, error) {
		args.Gas = hexutil.Uint64(gas)

		res, _, failed, err, vmErr := s.doCall(ctx, args, rpc.LatestBlockNumber, vm.Config{}, 0)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) {
				return false, nil, nil, nil // Special case, raise gas limit
			}
			return false, nil, err, nil // Bail out
		}
		if failed {
			return false, res, nil, vmErr
		}

		return true, nil, nil, nil
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		ok, _, err, _ := executable(mid)

		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle any more.
		if err != nil {
			return 0, err
		}

		if !ok {
			lo = mid
		} else {
			hi = mid
		}
	}

	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		ok, res, err, vmErr := executable(hi)
		if err != nil {
			return 0, err
		}

		if !ok {
			if vmErr != vm.ErrOutOfGas {
				if len(res) > 0 {
					return 0, newRevertError(res)
				}
				return 0, vmErr
			}

			// Otherwise, the specified gas cap is too low
			return 0, fmt.Errorf("gas required exceeds allowance (%d)", cap)						
		}
	}
	return hexutil.Uint64(hi), nil
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64         `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   error              `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

// formatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []vm.StructLog) []StructLogRes {
	formatted := make([]StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = StructLogRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   trace.Depth,
			Error:   trace.Err,
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = fmt.Sprintf("%x", math.PaddedBigBytes(stackValue, 32))
			}
			formatted[index].Stack = &stack
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
	}
	return formatted
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainAPI) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool, ctx context.Context) (map[string]interface{}, error) {
	head := b.Header() // copies the header once
	fields := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             b.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"totalDifficulty":  (*hexutil.Big)(s.b.GetTd(b.Hash())),
		"extraData":        hexutil.Bytes(head.Extra),
		"size":             hexutil.Uint64(b.Size()),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        (*hexutil.Big)(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
		"validators":       hexutil.Bytes(head.Validators),
		"validator":        hexutil.Bytes(head.Validator),
		"penalties":        hexutil.Bytes(head.Penalties),
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}

		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromBlockHash(b, tx.Hash()), nil
			}
		}

		txs := b.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range b.Transactions() {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}

	uncles := b.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes
	return fields, nil
}

// findNearestSignedBlock finds the nearest checkpoint from input block
func (s *PublicBlockChainAPI) findNearestSignedBlock(ctx context.Context, b *types.Block) *types.Block {
	if b.Number().Int64() <= 0 {
		return nil
	}

	blockNumber := b.Number().Uint64()
	signedBlockNumber := blockNumber + (common.MergeSignRange - (blockNumber % common.MergeSignRange))
	latestBlockNumber := s.b.CurrentBlock().Number()

	if signedBlockNumber >= latestBlockNumber.Uint64() || !s.b.ChainConfig().IsTIPSigning(b.Number()) {
		signedBlockNumber = blockNumber
	}

	// Get block epoc latest
	checkpointNumber := signedBlockNumber - (signedBlockNumber % s.b.ChainConfig().XDPoS.Epoch)
	checkpointBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(checkpointNumber))

	if checkpointBlock != nil {
		signedBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(signedBlockNumber))
		return signedBlock
	}

	return nil
}

/*
	findFinalityOfBlock return finality of a block
	Use blocksHashCache for to keep track - refer core/blockchain.go for more detail
*/
func (s *PublicBlockChainAPI) findFinalityOfBlock(ctx context.Context, b *types.Block, masternodes []common.Address) (uint, error) {
	engine, _ := s.b.GetEngine().(*XDPoS.XDPoS)
	signedBlock := s.findNearestSignedBlock(ctx, b)

	if signedBlock == nil {
		return 0, nil
	}

	signedBlocksHash := s.b.GetBlocksHashCache(signedBlock.Number().Uint64())

	// there is no cache for this block's number
	// return the number(signers) / number(masternode) * 100 if this block is on canonical path
	// else return 0 for fork path
	if signedBlocksHash == nil {
		if !s.b.AreTwoBlockSamePath(signedBlock.Hash(), b.Hash()) {
			return 0, nil
		}

		blockSigners, err := s.getSigners(ctx, signedBlock, engine)
		if blockSigners == nil {
			return 0, err
		}

		return uint(100 * len(blockSigners) / len(masternodes)), nil
	}

	/*
		With Hashes cache - we can track all chain's path
		back to current's block number by parent's Hash
		If found the current block so the finality = signedBlock's finality
		else return 0
	*/

	var signedBlockSamePath common.Hash

	for count := 0; count < len(signedBlocksHash); count++ {
		blockHash := signedBlocksHash[count]
		if s.b.AreTwoBlockSamePath(blockHash, b.Hash()) {
			signedBlockSamePath = blockHash
			break
		}
	}

	// return 0 if not same path with any signed block
	if len(signedBlockSamePath) == 0 {
		return 0, nil
	}

	// get signers and return finality
	samePathSignedBlock, err := s.b.GetBlock(ctx, signedBlockSamePath)
	if samePathSignedBlock == nil {
		return 0, err
	}

	blockSigners, err := s.getSigners(ctx, samePathSignedBlock, engine)
	if blockSigners == nil {
		return 0, err
	}

	return uint(100 * len(blockSigners) / len(masternodes)), nil
}

/*
	Extract signers from block
*/
func (s *PublicBlockChainAPI) getSigners(ctx context.Context, block *types.Block, engine *XDPoS.XDPoS) ([]common.Address, error) {
	var err error
	var filterSigners []common.Address
	var signers []common.Address
	blockNumber := block.Number().Uint64()

	// Get block epoc latest.
	checkpointNumber := blockNumber - (blockNumber % s.b.ChainConfig().XDPoS.Epoch)
	checkpointBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(checkpointNumber))

	masternodes := engine.GetMasternodesFromCheckpointHeader(checkpointBlock.Header(), blockNumber, s.b.ChainConfig().XDPoS.Epoch)
	signers, err = GetSignersFromBlocks(s.b, block.NumberU64(), block.Hash(), masternodes)
	if err != nil {
		log.Error("Fail to get signers from block signer SC.", "error", err)
		return nil, err
	}
	validator, _ := engine.RecoverValidator(block.Header())
	creator, _ := engine.RecoverSigner(block.Header())
	signers = append(signers, validator)
	signers = append(signers, creator)

	for _, masternode := range masternodes {
		for _, signer := range signers {
			if signer == masternode {
				filterSigners = append(filterSigners, masternode)
				break
			}
		}
	}
	return filterSigners, nil
}

func (s *PublicBlockChainAPI) rpcOutputBlockSigners(b *types.Block, ctx context.Context, masternodes []common.Address) ([]common.Address, error) {
	_, err := s.b.GetIPCClient()
	if err != nil {
		log.Error("Fail to connect IPC client for block status", "error", err)
		return []common.Address{}, err
	}

	engine, ok := s.b.GetEngine().(*XDPoS.XDPoS)
	if !ok {
		log.Error("Undefined XDPoS consensus engine")
		return []common.Address{}, nil
	}

	signedBlock := s.findNearestSignedBlock(ctx, b)
	if signedBlock == nil {
		return []common.Address{}, nil
	}

	return s.getSigners(ctx, signedBlock, engine)
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash     `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex hexutil.Uint    `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCTransaction {
	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()

	result := &RPCTransaction{
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = hexutil.Uint(index)
	}
	return result
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	return newRPCTransaction(tx, common.Hash{}, 0, 0)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, index uint64) *RPCTransaction {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], b.Hash(), b.NumberU64(), index)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *types.Block, index uint64) hexutil.Bytes {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := rlp.EncodeToBytes(txs[index])
	return blob
}

// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockHash(b *types.Block, hash common.Hash) *RPCTransaction {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == hash {
			return newRPCTransactionFromBlockIndex(b, uint64(idx))
		}
	}
	return nil
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	b         Backend
	nonceLock *AddrLocker
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicXDCXTransactionPoolAPI struct {
	b         Backend
	nonceLock *AddrLocker
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(b Backend, nonceLock *AddrLocker) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{b, nonceLock}
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicXDCXTransactionPoolAPI(b Backend, nonceLock *AddrLocker) *PublicXDCXTransactionPoolAPI {
	return &PublicXDCXTransactionPoolAPI{b, nonceLock}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockNumberAndIndex returns the bytes of the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockHashAndIndex returns the bytes of the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*hexutil.Uint64, error) {
	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) *RPCTransaction {
	// Try to return an already finalized transaction
	if tx, blockHash, blockNumber, index := core.GetTransaction(s.b.ChainDb(), hash); tx != nil {
		return newRPCTransaction(tx, blockHash, blockNumber, index)
	}
	// No finalized transaction, try to retrieve it from the pool
	if tx := s.b.GetPoolTransaction(hash); tx != nil {
		return newRPCPendingTransaction(tx)
	}
	// Transaction unknown, return as such
	return nil
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (s *PublicTransactionPoolAPI) GetRawTransactionByHash(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	var tx *types.Transaction

	// Retrieve a finalized transaction, or a pooled otherwise
	if tx, _, _, _ = core.GetTransaction(s.b.ChainDb(), hash); tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			// Transaction not found anywhere, abort
			return nil, nil
		}
	}
	// Serialize to RLP and return
	return rlp.EncodeToBytes(tx)
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockHash, blockNumber, index := core.GetTransaction(s.b.ChainDb(), hash)
	if tx == nil {
		return nil, nil
	}
	receipts, err := s.b.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if len(receipts) <= int(index) {
		return nil, nil
	}
	receipt := receipts[index]

	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
	}

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolAPI) sign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	var chainID *big.Int
	if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number()) {
		chainID = config.ChainId
	}
	return wallet.SignTx(account, tx, chainID)
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

// setDefaults is a helper function that fills in default values for unspecified tx fields.
func (args *SendTxArgs) setDefaults(ctx context.Context, b Backend) error {
	if args.Gas == nil {
		args.Gas = new(hexutil.Uint64)
		*(*uint64)(args.Gas) = 90000
	}
	if args.GasPrice == nil {
		price, err := b.SuggestPrice(ctx)
		if err != nil {
			return err
		}
		args.GasPrice = (*hexutil.Big)(price)
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.From)
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`Both "data" and "input" are set and not equal. Please use "input" to pass transaction call data.`)
	}
	if args.To == nil {
		// Contract creation
		var input []byte
		if args.Data != nil {
			input = *args.Data
		} else if args.Input != nil {
			input = *args.Input
		}
		if len(input) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
	}
	return nil
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Data != nil {
		input = *args.Data
	} else if args.Input != nil {
		input = *args.Input
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
}

// submitTransaction is a helper function that submits tx to txPool and logs a message.
func submitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	if tx.To() != nil && tx.IsSpecialTransaction() {
		return common.Hash{}, errors.New("Dont allow transaction sent to BlockSigners & RandomizeSMC smart contract via API")
	}
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	if tx.To() == nil {
		signer := types.MakeSigner(b.ChainConfig(), b.CurrentBlock().Number())
		from, err := types.Sender(signer, tx)
		if err != nil {
			return common.Hash{}, err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Trace("Submitted contract creation", "fullhash", tx.Hash().Hex(), "contract", addr.Hex())
	} else {
		log.Trace("Submitted transaction", "fullhash", tx.Hash().Hex(), "recipient", tx.To())
	}
	return tx.Hash(), nil
}

// submitTransaction is a helper function that submits tx to txPool and logs a message.
func submitOrderTransaction(ctx context.Context, b Backend, tx *types.OrderTransaction) (common.Hash, error) {

	if err := b.SendOrderTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	return tx.Hash(), nil
}

// submitLendingTransaction is a helper function that submits tx to txPool and logs a message.
func submitLendingTransaction(ctx context.Context, b Backend, tx *types.LendingTransaction) (common.Hash, error) {

	if err := b.SendLendingTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	return tx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolAPI) SendTransaction(ctx context.Context, args SendTxArgs) (common.Hash, error) {

	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.From}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.From)
		defer s.nonceLock.UnlockAddr(args.From)
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.toTransaction()

	var chainID *big.Int
	if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number()) {
		chainID = config.ChainId
	}
	signed, err := wallet.SignTx(account, tx, chainID)
	if err != nil {
		return common.Hash{}, err
	}
	return submitTransaction(ctx, s.b, signed)
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	return submitTransaction(ctx, s.b, tx)
}

// SendOrderRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicXDCXTransactionPoolAPI) SendOrderRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.OrderTransaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	return submitOrderTransaction(ctx, s.b, tx)
}

// SendLendingRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicXDCXTransactionPoolAPI) SendLendingRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.LendingTransaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, err
	}
	return submitLendingTransaction(ctx, s.b, tx)
}

// GetOrderTxMatchByHash returns the bytes of the transaction for the given hash.
func (s *PublicXDCXTransactionPoolAPI) GetOrderTxMatchByHash(ctx context.Context, hash common.Hash) ([]*tradingstate.OrderItem, error) {
	var tx *types.Transaction
	orders := []*tradingstate.OrderItem{}
	if tx, _, _, _ = core.GetTransaction(s.b.ChainDb(), hash); tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			return []*tradingstate.OrderItem{}, nil
		}
	}

	batch, err := tradingstate.DecodeTxMatchesBatch(tx.Data())
	if err != nil {
		return []*tradingstate.OrderItem{}, err
	}
	for _, txMatch := range batch.Data {
		order, err := txMatch.DecodeOrder()
		if err != nil {
			return []*tradingstate.OrderItem{}, err
		}
		orders = append(orders, order)
	}
	return orders, nil

}

// GetOrderPoolContent return pending, queued content
func (s *PublicXDCXTransactionPoolAPI) GetOrderPoolContent(ctx context.Context) interface{} {
	pendingOrders := []*tradingstate.OrderItem{}
	queuedOrders := []*tradingstate.OrderItem{}
	pending, queued := s.b.OrderTxPoolContent()

	for _, txs := range pending {
		for _, tx := range txs {
			V, R, S := tx.Signature()
			order := &tradingstate.OrderItem{
				Nonce:           big.NewInt(int64(tx.Nonce())),
				Quantity:        tx.Quantity(),
				Price:           tx.Price(),
				ExchangeAddress: tx.ExchangeAddress(),
				UserAddress:     tx.UserAddress(),
				BaseToken:       tx.BaseToken(),
				QuoteToken:      tx.QuoteToken(),
				Status:          tx.Status(),
				Side:            tx.Side(),
				Type:            tx.Type(),
				Hash:            tx.OrderHash(),
				OrderID:         tx.OrderID(),
				Signature: &tradingstate.Signature{
					V: byte(V.Uint64()),
					R: common.BigToHash(R),
					S: common.BigToHash(S),
				},
			}
			pendingOrders = append(pendingOrders, order)
		}
	}

	for _, txs := range queued {
		for _, tx := range txs {
			V, R, S := tx.Signature()
			order := &tradingstate.OrderItem{
				Nonce:           big.NewInt(int64(tx.Nonce())),
				Quantity:        tx.Quantity(),
				Price:           tx.Price(),
				ExchangeAddress: tx.ExchangeAddress(),
				UserAddress:     tx.UserAddress(),
				BaseToken:       tx.BaseToken(),
				QuoteToken:      tx.QuoteToken(),
				Status:          tx.Status(),
				Side:            tx.Side(),
				Type:            tx.Type(),
				Hash:            tx.OrderHash(),
				OrderID:         tx.OrderID(),
				Signature: &tradingstate.Signature{
					V: byte(V.Uint64()),
					R: common.BigToHash(R),
					S: common.BigToHash(S),
				},
			}
			queuedOrders = append(pendingOrders, order)
		}
	}

	return map[string]interface{}{
		"pending": pendingOrders,
		"queued":  queuedOrders,
	}
}

// GetOrderStats return pending, queued length
func (s *PublicXDCXTransactionPoolAPI) GetOrderStats(ctx context.Context) interface{} {
	pending, queued := s.b.OrderStats()
	return map[string]interface{}{
		"pending": pending,
		"queued":  queued,
	}
}

// OrderMsg struct
type OrderMsg struct {
	AccountNonce    hexutil.Uint64 `json:"nonce"    gencodec:"required"`
	Quantity        hexutil.Big    `json:"quantity,omitempty"`
	Price           hexutil.Big    `json:"price,omitempty"`
	ExchangeAddress common.Address `json:"exchangeAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	BaseToken       common.Address `json:"baseToken,omitempty"`
	QuoteToken      common.Address `json:"quoteToken,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	OrderID         hexutil.Uint64 `json:"orderid,omitempty"`
	// Signature values
	V hexutil.Big `json:"v" gencodec:"required"`
	R hexutil.Big `json:"r" gencodec:"required"`
	S hexutil.Big `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

// LendingMsg api message for lending
type LendingMsg struct {
	AccountNonce    hexutil.Uint64 `json:"nonce"    gencodec:"required"`
	Quantity        hexutil.Big    `json:"quantity,omitempty"`
	RelayerAddress  common.Address `json:"relayerAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	CollateralToken common.Address `json:"collateralToken,omitempty"`
	AutoTopUp       bool           `json:"autoTopUp,omitempty"`
	LendingToken    common.Address `json:"lendingToken,omitempty"`
	Term            hexutil.Uint64 `json:"term,omitempty"`
	Interest        hexutil.Uint64 `json:"interest,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	LendingId       hexutil.Uint64 `json:"lendingId,omitempty"`
	LendingTradeId  hexutil.Uint64 `json:"tradeId,omitempty"`
	ExtraData       string         `json:"extraData,omitempty"`

	// Signature values
	V hexutil.Big `json:"v" gencodec:"required"`
	R hexutil.Big `json:"r" gencodec:"required"`
	S hexutil.Big `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

type PriceVolume struct {
	Price  *big.Int `json:"price,omitempty"`
	Volume *big.Int `json:"volume,omitempty"`
}

type InterestVolume struct {
	Interest *big.Int `json:"interest,omitempty"`
	Volume   *big.Int `json:"volume,omitempty"`
}

// SendOrder will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicXDCXTransactionPoolAPI) SendOrder(ctx context.Context, msg OrderMsg) (common.Hash, error) {
	tx := types.NewOrderTransaction(uint64(msg.AccountNonce), msg.Quantity.ToInt(), msg.Price.ToInt(), msg.ExchangeAddress, msg.UserAddress, msg.BaseToken, msg.QuoteToken, msg.Status, msg.Side, msg.Type, msg.Hash, uint64(msg.OrderID))
	tx = tx.ImportSignature(msg.V.ToInt(), msg.R.ToInt(), msg.S.ToInt())
	return submitOrderTransaction(ctx, s.b, tx)
}

// SendLending will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicXDCXTransactionPoolAPI) SendLending(ctx context.Context, msg LendingMsg) (common.Hash, error) {
	tx := types.NewLendingTransaction(uint64(msg.AccountNonce), msg.Quantity.ToInt(), uint64(msg.Interest), uint64(msg.Term), msg.RelayerAddress, msg.UserAddress, msg.LendingToken, msg.CollateralToken, msg.AutoTopUp, msg.Status, msg.Side, msg.Type, msg.Hash, uint64(msg.LendingId), uint64(msg.LendingTradeId), msg.ExtraData)
	tx = tx.ImportSignature(msg.V.ToInt(), msg.R.ToInt(), msg.S.ToInt())
	return submitLendingTransaction(ctx, s.b, tx)
}

// GetOrderCount returns the number of transactions the given address has sent for the given block number
func (s *PublicXDCXTransactionPoolAPI) GetOrderCount(ctx context.Context, addr common.Address) (*hexutil.Uint64, error) {

	nonce, err := s.b.GetOrderNonce(addr.Hash())
	if err != nil {
		return (*hexutil.Uint64)(&nonce), err
	}
	return (*hexutil.Uint64)(&nonce), err
}

func (s *PublicXDCXTransactionPoolAPI) GetBestBid(ctx context.Context, baseToken, quoteToken common.Address) (PriceVolume, error) {

	result := PriceVolume{}
	block := s.b.CurrentBlock()
	if block == nil {
		return result, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return result, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return result, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return result, err
	}
	result.Price, result.Volume = XDCxState.GetBestBidPrice(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if result.Price.Sign() == 0 {
		return result, errors.New("Bid tree not found")
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetBestAsk(ctx context.Context, baseToken, quoteToken common.Address) (PriceVolume, error) {
	result := PriceVolume{}
	block := s.b.CurrentBlock()
	if block == nil {
		return result, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return result, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return result, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return result, err
	}
	result.Price, result.Volume = XDCxState.GetBestAskPrice(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if result.Price.Sign() == 0 {
		return result, errors.New("Ask tree not found")
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetBidTree(ctx context.Context, baseToken, quoteToken common.Address) (map[*big.Int]tradingstate.DumpOrderList, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := XDCxState.DumpBidTrie(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetPrice(ctx context.Context, baseToken, quoteToken common.Address) (*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	price := XDCxState.GetLastPrice(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if price == nil || price.Sign() == 0 {
		return common.Big0, errors.New("Order book's price not found")
	}
	return price, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLastEpochPrice(ctx context.Context, baseToken, quoteToken common.Address) (*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	price := XDCxState.GetMediumPriceBeforeEpoch(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if price == nil || price.Sign() == 0 {
		return common.Big0, errors.New("Order book's price not found")
	}
	return price, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetCurrentEpochPrice(ctx context.Context, baseToken, quoteToken common.Address) (*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	price, _ := XDCxState.GetMediumPriceAndTotalAmount(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if price == nil || price.Sign() == 0 {
		return common.Big0, errors.New("Order book's price not found")
	}
	return price, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetAskTree(ctx context.Context, baseToken, quoteToken common.Address) (map[*big.Int]tradingstate.DumpOrderList, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := XDCxState.DumpAskTrie(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetOrderById(ctx context.Context, baseToken, quoteToken common.Address, orderId uint64) (interface{}, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderId))
	orderitem := XDCxState.GetOrder(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken), orderIdHash)
	if orderitem.Quantity == nil || orderitem.Quantity.Sign() == 0 {
		return nil, errors.New("Order not found")
	}
	return orderitem, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetTradingOrderBookInfo(ctx context.Context, baseToken, quoteToken common.Address) (*tradingstate.DumpOrderBookInfo, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := XDCxState.DumpOrderBookInfo(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLiquidationPriceTree(ctx context.Context, baseToken, quoteToken common.Address) (map[*big.Int]tradingstate.DumpLendingBook, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := XDCxState.DumpLiquidationPriceTrie(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetInvestingTree(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]lendingstate.DumpOrderList, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.DumpInvestingTrie(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetBorrowingTree(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]lendingstate.DumpOrderList, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.DumpBorrowingTrie(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLendingOrderBookInfo(tx context.Context, lendingToken common.Address, term uint64) (*lendingstate.DumpOrderBookInfo, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.DumpOrderBookInfo(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) getLendingOrderTree(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]lendingstate.LendingItem, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.DumpLendingOrderTrie(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLendingTradeTree(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]lendingstate.LendingTrade, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.DumpLendingTradeTrie(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLiquidationTimeTree(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]lendingstate.DumpOrderList, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.DumpLiquidationTimeTrie(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLendingOrderCount(ctx context.Context, addr common.Address) (*hexutil.Uint64, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	nonce := lendingState.GetNonce(addr.Hash())
	return (*hexutil.Uint64)(&nonce), err
}

func (s *PublicXDCXTransactionPoolAPI) GetBestInvesting(ctx context.Context, lendingToken common.Address, term uint64) (InterestVolume, error) {
	result := InterestVolume{}
	block := s.b.CurrentBlock()
	if block == nil {
		return result, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return result, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return result, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return result, err
	}
	result.Interest, result.Volume = lendingState.GetBestInvestingRate(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetBestBorrowing(ctx context.Context, lendingToken common.Address, term uint64) (InterestVolume, error) {
	result := InterestVolume{}
	block := s.b.CurrentBlock()
	if block == nil {
		return result, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return result, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return result, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return result, err
	}
	result.Interest, result.Volume = lendingState.GetBestBorrowRate(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetBids(ctx context.Context, baseToken, quoteToken common.Address) (map[*big.Int]*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := XDCxState.GetBids(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetAsks(ctx context.Context, baseToken, quoteToken common.Address) (map[*big.Int]*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	XDCxService := s.b.XDCxService()
	if XDCxService == nil {
		return nil, errors.New("XDCX service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	XDCxState, err := XDCxService.GetTradingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := XDCxState.GetAsks(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetInvests(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.GetInvestings(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetBorrows(ctx context.Context, lendingToken common.Address, term uint64) (map[*big.Int]*big.Int, error) {
	block := s.b.CurrentBlock()
	if block == nil {
		return nil, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return nil, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return nil, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return nil, err
	}
	result, err := lendingState.GetBorrowings(lendingstate.GetLendingOrderBookHash(lendingToken, term))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetLendingTxMatchByHash returns lendingItems which have been processed at tx of the given txhash
func (s *PublicXDCXTransactionPoolAPI) GetLendingTxMatchByHash(ctx context.Context, hash common.Hash) ([]*lendingstate.LendingItem, error) {
	var tx *types.Transaction
	if tx, _, _, _ = core.GetTransaction(s.b.ChainDb(), hash); tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			return []*lendingstate.LendingItem{}, nil
		}
	}

	batch, err := lendingstate.DecodeTxLendingBatch(tx.Data())
	if err != nil {
		return []*lendingstate.LendingItem{}, err
	}
	return batch.Data, nil
}

// GetLiquidatedTradesByTxHash returns trades which closed by XDCX protocol at the tx of the give hash
func (s *PublicXDCXTransactionPoolAPI) GetLiquidatedTradesByTxHash(ctx context.Context, hash common.Hash) (lendingstate.FinalizedResult, error) {
	var tx *types.Transaction
	if tx, _, _, _ = core.GetTransaction(s.b.ChainDb(), hash); tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			return lendingstate.FinalizedResult{}, nil
		}
	}

	finalizedResult, err := lendingstate.DecodeFinalizedResult(tx.Data())
	if err != nil {
		return lendingstate.FinalizedResult{}, err
	}
	finalizedResult.TxHash = hash
	return finalizedResult, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLendingOrderById(ctx context.Context, lendingToken common.Address, term uint64, orderId uint64) (lendingstate.LendingItem, error) {
	lendingItem := lendingstate.LendingItem{}
	block := s.b.CurrentBlock()
	if block == nil {
		return lendingItem, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return lendingItem, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return lendingItem, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return lendingItem, err
	}
	lendingOrderBook := lendingstate.GetLendingOrderBookHash(lendingToken, term)
	orderIdHash := common.BigToHash(new(big.Int).SetUint64(orderId))
	lendingItem = lendingState.GetLendingOrder(lendingOrderBook, orderIdHash)
	if lendingItem.LendingId != orderId {
		return lendingItem, errors.New("Lending Item not found")
	}
	return lendingItem, nil
}

func (s *PublicXDCXTransactionPoolAPI) GetLendingTradeById(ctx context.Context, lendingToken common.Address, term uint64, tradeId uint64) (lendingstate.LendingTrade, error) {
	lendingItem := lendingstate.LendingTrade{}
	block := s.b.CurrentBlock()
	if block == nil {
		return lendingItem, errors.New("Current block not found")
	}
	lendingService := s.b.LendingService()
	if lendingService == nil {
		return lendingItem, errors.New("XDCX Lending service not found")
	}
	author, err := s.b.GetEngine().Author(block.Header())
	if err != nil {
		return lendingItem, err
	}
	lendingState, err := lendingService.GetLendingState(block, author)
	if err != nil {
		return lendingItem, err
	}
	lendingOrderBook := lendingstate.GetLendingOrderBookHash(lendingToken, term)
	tradeIdHash := common.BigToHash(new(big.Int).SetUint64(tradeId))
	lendingItem = lendingState.GetLendingTrade(lendingOrderBook, tradeIdHash)
	if lendingItem.TradeId != tradeId {
		return lendingItem, errors.New("Lending Item not found")
	}
	return lendingItem, nil
}

// Sign calculates an ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message).
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The account associated with addr must be unlocked.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
func (s *PublicTransactionPoolAPI) Sign(addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Sign the requested hash with the wallet
	signature, err := wallet.SignHash(account, signHash(data))
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, err
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolAPI) SignTransaction(ctx context.Context, args SendTxArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		return nil, fmt.Errorf("gas not specified")
	}
	if args.GasPrice == nil {
		return nil, fmt.Errorf("gasPrice not specified")
	}
	if args.Nonce == nil {
		return nil, fmt.Errorf("nonce not specified")
	}
	if err := args.setDefaults(ctx, s.b); err != nil {
		return nil, err
	}
	tx, err := s.sign(args.From, args.toTransaction())
	if err != nil {
		return nil, err
	}
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool and have a from address that is one of
// the accounts this node manages.
func (s *PublicTransactionPoolAPI) PendingTransactions() ([]*RPCTransaction, error) {
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return nil, err
	}

	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if tx.Protected() {
			signer = types.NewEIP155Signer(tx.ChainId())
		}
		from, _ := types.Sender(signer, tx)
		if _, err := s.b.AccountManager().Find(accounts.Account{Address: from}); err == nil {
			transactions = append(transactions, newRPCPendingTransaction(tx))
		}
	}
	return transactions, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolAPI) Resend(ctx context.Context, sendArgs SendTxArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	if sendArgs.Nonce == nil {
		return common.Hash{}, fmt.Errorf("missing transaction nonce in transaction spec")
	}
	if err := sendArgs.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	matchTx := sendArgs.toTransaction()
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return common.Hash{}, err
	}

	for _, p := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if p.Protected() {
			signer = types.NewEIP155Signer(p.ChainId())
		}
		wantSigHash := signer.Hash(matchTx)

		if pFrom, err := types.Sender(signer, p); err == nil && pFrom == sendArgs.From && signer.Hash(p) == wantSigHash {
			// Match. Re-sign and send the transaction.
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			signedTx, err := s.sign(sendArgs.From, sendArgs.toTransaction())
			if err != nil {
				return common.Hash{}, err
			}
			if err = s.b.SendTx(ctx, signedTx); err != nil {
				return common.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}

	return common.Hash{}, fmt.Errorf("Transaction %#x not found", matchTx.Hash())
}

// PublicDebugAPI is the collection of Ethereum APIs exposed over the public
// debugging endpoint.
type PublicDebugAPI struct {
	b Backend
}

// NewPublicDebugAPI creates a new API definition for the public debug methods
// of the Ethereum service.
func NewPublicDebugAPI(b Backend) *PublicDebugAPI {
	return &PublicDebugAPI{b: b}
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *PublicDebugAPI) GetBlockRlp(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *PublicDebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return block.String(), nil
}

// SeedHash retrieves the seed hash of a block.
func (api *PublicDebugAPI) SeedHash(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return fmt.Sprintf("0x%x", ethash.SeedHash(number)), nil
}

// PrivateDebugAPI is the collection of Ethereum APIs exposed over the private
// debugging endpoint.
type PrivateDebugAPI struct {
	b Backend
}

// NewPrivateDebugAPI creates a new API definition for the private debug methods
// of the Ethereum service.
func NewPrivateDebugAPI(b Backend) *PrivateDebugAPI {
	return &PrivateDebugAPI{b: b}
}

// ChaindbProperty returns leveldb properties of the chain database.
func (api *PrivateDebugAPI) ChaindbProperty(property string) (string, error) {
	ldb, ok := api.b.ChainDb().(interface {
		LDB() *leveldb.DB
	})
	if !ok {
		return "", fmt.Errorf("chaindbProperty does not work for memory databases")
	}
	if property == "" {
		property = "leveldb.stats"
	} else if !strings.HasPrefix(property, "leveldb.") {
		property = "leveldb." + property
	}
	return ldb.LDB().GetProperty(property)
}

func (api *PrivateDebugAPI) ChaindbCompact() error {
	ldb, ok := api.b.ChainDb().(interface {
		LDB() *leveldb.DB
	})
	if !ok {
		return fmt.Errorf("chaindbCompact does not work for memory databases")
	}
	for b := byte(0); b < 255; b++ {
		log.Info("Compacting chain database", "range", fmt.Sprintf("0x%0.2X-0x%0.2X", b, b+1))
		err := ldb.LDB().CompactRange(util.Range{Start: []byte{b}, Limit: []byte{b + 1}})
		if err != nil {
			log.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *PrivateDebugAPI) SetHead(number hexutil.Uint64) {
	api.b.SetHead(uint64(number))
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewPublicNetAPI creates a new net API instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion uint64) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(s.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}

func GetSignersFromBlocks(b Backend, blockNumber uint64, blockHash common.Hash, masternodes []common.Address) ([]common.Address, error) {
	var addrs []common.Address
	mapMN := map[common.Address]bool{}
	for _, node := range masternodes {
		mapMN[node] = true
	}
	signer := types.MakeSigner(b.ChainConfig(), new(big.Int).SetUint64(blockNumber))
	if engine, ok := b.GetEngine().(*XDPoS.XDPoS); ok {
		limitNumber := blockNumber + common.LimitTimeFinality
		currentNumber := b.CurrentBlock().NumberU64()
		if limitNumber > currentNumber {
			limitNumber = currentNumber
		}
		for i := blockNumber + 1; i <= limitNumber; i++ {
			header, err := b.HeaderByNumber(nil, rpc.BlockNumber(i))
			if err != nil {
				return addrs, err
			}
			blockData, err := b.BlockByNumber(nil, rpc.BlockNumber(i))
			signTxs := engine.CacheSigningTxs(header.Hash(), blockData.Transactions())
			for _, signtx := range signTxs {
				blkHash := common.BytesToHash(signtx.Data()[len(signtx.Data())-32:])
				from, _ := types.Sender(signer, signtx)
				if blkHash == blockHash && mapMN[from] {
					addrs = append(addrs, from)
					delete(mapMN, from)
				}
			}
			if len(mapMN) == 0 {
				break
			}
		}
	}
	return addrs, nil
}

// GetStakerROI Estimate ROI for stakers using the last epoc reward
// then multiple by epoch per year, if the address is not masternode of last epoch - return 0
// Formular:
// 		ROI = average_latest_epoch_reward_for_voters*number_of_epoch_per_year/latest_total_cap*100
func (s *PublicBlockChainAPI) GetStakerROI() float64 {
	blockNumber := s.b.CurrentBlock().Number().Uint64()
	lastCheckpointNumber := blockNumber - (blockNumber % s.b.ChainConfig().XDPoS.Epoch) - s.b.ChainConfig().XDPoS.Epoch // calculate for 2 epochs ago
	totalCap := new(big.Int).SetUint64(0)

	mastersCap := s.b.GetMasternodesCap(lastCheckpointNumber)
	if mastersCap == nil {
		return 0
	}

	masternodeReward := new(big.Int).Mul(new(big.Int).SetUint64(s.b.ChainConfig().XDPoS.Reward), new(big.Int).SetUint64(params.Ether))

	for _, cap := range mastersCap {
		totalCap.Add(totalCap, cap)
	}

	holderReward := new(big.Int).Div(masternodeReward, new(big.Int).SetUint64(2))
	EpochPerYear := 365 * 86400 / s.b.GetEpochDuration().Uint64()
	voterRewardAYear := new(big.Int).Mul(holderReward, new(big.Int).SetUint64(EpochPerYear))
	return 100.0 / float64(totalCap.Div(totalCap, voterRewardAYear).Uint64())
}

// GetStakerROIMasternode Estimate ROI for stakers of a specific masternode using the last epoc reward
// then multiple by epoch per year, if the address is not masternode of last epoch - return 0
// Formular:
// 		ROI = latest_epoch_reward_for_voters*number_of_epoch_per_year/latest_total_cap*100
func (s *PublicBlockChainAPI) GetStakerROIMasternode(masternode common.Address) float64 {
	votersReward := s.b.GetVotersRewards(masternode)
	if votersReward == nil {
		return 0
	}

	masternodeReward := new(big.Int).SetUint64(0) // this includes all reward for this masternode
	voters := []common.Address{}
	for voter, reward := range votersReward {
		voters = append(voters, voter)
		masternodeReward.Add(masternodeReward, reward)
	}

	blockNumber := s.b.CurrentBlock().Number().Uint64()
	lastCheckpointNumber := blockNumber - blockNumber%s.b.ChainConfig().XDPoS.Epoch
	totalCap := new(big.Int).SetUint64(0)
	votersCap := s.b.GetVotersCap(new(big.Int).SetUint64(lastCheckpointNumber), masternode, voters)

	for _, cap := range votersCap {
		totalCap.Add(totalCap, cap)
	}

	// holder reward = 50% total reward of a masternode
	holderReward := new(big.Int).Div(masternodeReward, new(big.Int).SetUint64(2))
	EpochPerYear := 365 * 86400 / s.b.GetEpochDuration().Uint64()
	voterRewardAYear := new(big.Int).Mul(holderReward, new(big.Int).SetUint64(EpochPerYear))

	return 100.0 / float64(totalCap.Div(totalCap, voterRewardAYear).Uint64())
}
