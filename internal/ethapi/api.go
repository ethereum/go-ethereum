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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/ethash"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/net/context"
)

const defaultGas = uint64(90000)

var ErrNoPendingState = errors.New("API backend does not provide access to pending state")

// PublicEthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumAPI struct {
	b Backend
}

// NewPublicEthereumAPI creates a new Etheruem protocol API.
func NewPublicEthereumAPI(b Backend) *PublicEthereumAPI {
	return &PublicEthereumAPI{b}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicEthereumAPI) GasPrice(ctx context.Context) (*big.Int, error) {
	return s.b.SuggestGasPrice(ctx)
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *PublicEthereumAPI) ProtocolVersion() *rpc.HexNumber {
	return rpc.NewHexNumber(s.b.ProtocolVersion())
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *PublicEthereumAPI) Syncing(ctx context.Context) (interface{}, error) {
	progress, err := s.b.SyncProgress(ctx)
	if err != nil {
		return nil, err
	}

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock": rpc.NewHexNumber(progress.StartingBlock),
		"currentBlock":  rpc.NewHexNumber(progress.CurrentBlock),
		"highestBlock":  rpc.NewHexNumber(progress.HighestBlock),
		"pulledStates":  rpc.NewHexNumber(progress.PulledStates),
		"knownStates":   rpc.NewHexNumber(progress.KnownStates),
	}, nil
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
func (s *PublicAccountAPI) Accounts() []accounts.Account {
	return s.am.Accounts()
}

// PrivateAccountAPI provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts. Some methods accept
// passwords and are therefore considered private by default.
type PrivateAccountAPI struct {
	am *accounts.Manager
	b  Backend
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(b Backend) *PrivateAccountAPI {
	return &PrivateAccountAPI{
		am: b.AccountManager(),
		b:  b,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() []common.Address {
	accounts := s.am.Accounts()
	addresses := make([]common.Address, len(accounts))
	for i, acc := range accounts {
		addresses[i] = acc.Address
	}
	return addresses
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (common.Address, error) {
	acc, err := s.am.NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (s *PrivateAccountAPI) ImportRawKey(privkey string, password string) (common.Address, error) {
	hexkey, err := hex.DecodeString(privkey)
	if err != nil {
		return common.Address{}, err
	}

	acc, err := s.am.ImportECDSA(crypto.ToECDSA(hexkey), password)
	return acc.Address, err
}

// UnlockAccount will unlock the account associated with the given address with
// the given password for duration seconds. If duration is nil it will use a
// default of 300 seconds. It returns an indication if the account was unlocked.
func (s *PrivateAccountAPI) UnlockAccount(addr common.Address, password string, duration *rpc.HexNumber) (bool, error) {
	if duration == nil {
		duration = rpc.NewHexNumber(300)
	}
	a := accounts.Account{Address: addr}
	d := time.Duration(duration.Int64()) * time.Second
	if err := s.am.TimedUnlock(a, password, d); err != nil {
		return false, err
	}
	return true, nil
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr common.Address) bool {
	return s.am.Lock(addr) == nil
}

// SendTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails.
func (s *PrivateAccountAPI) SendTransaction(ctx context.Context, args SendTxArgs, passwd string) (common.Hash, error) {
	var err error
	args, err = prepareSendTxArgs(ctx, args, s.b)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		nonce, err := s.b.PendingNonceAt(ctx, args.From)
		if err != nil {
			return common.Hash{}, err
		}
		args.Nonce = rpc.NewHexNumber(nonce)
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	head, err := s.b.HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, err
	}
	signer := types.MakeSigner(s.b.ChainConfig(), head.Number)
	signature, err := s.am.SignWithPassphrase(args.From, passwd, signer.Hash(tx).Bytes())
	if err != nil {
		return common.Hash{}, err
	}

	return submitTransaction(ctx, s.b, tx, signer, signature)
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from. The hash is calulcated with:
// keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
func signHash(message string) []byte {
	data := common.FromHex(message)
	// Give context to the signed message. This prevents an adversery to sign a tx.
	// It has no cryptographic purpose.
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	// Always hash, this prevents choosen plaintext attacks that can extract key information
	return crypto.Keccak256([]byte(msg))
}

// Sign calculates an Ethereum ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message))
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_sign
func (s *PrivateAccountAPI) Sign(ctx context.Context, message string, addr common.Address, passwd string) (string, error) {
	hash := signHash(message)
	signature, err := s.b.AccountManager().SignWithPassphrase(addr, passwd, hash)
	if err != nil {
		return "0x", err
	}
	return common.ToHex(signature), nil
}

// EcRecover returns the address for the account that was used to create the signature.
// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
// the address of:
// hash = keccak256("\x19Ethereum Signed Message:\n"${message length}${message})
// addr = ecrecover(hash, signature)
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
func (s *PrivateAccountAPI) EcRecover(ctx context.Context, message string, signature string) (common.Address, error) {
	var (
		hash = signHash(message)
		sig  = common.FromHex(signature)
	)

	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}

	// see crypto.Ecrecover description
	if sig[64] == 27 || sig[64] == 28 {
		sig[64] -= 27
	}

	rpk, err := crypto.Ecrecover(hash, sig)
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

// NewPublicBlockChainAPI creates a new Etheruem blockchain API.
func NewPublicBlockChainAPI(b Backend) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{b}
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) BlockNumber() *big.Int {
	header, _ := s.b.HeaderByNumber(context.Background(), nil) // latest header should always be available
	return header.Number
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(ctx context.Context, blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := blockByNumber(ctx, s.b, blockNr)
	if block != nil {
		response, err := s.rpcOutputBlock(ctx, block, true, fullTx)
		if err == nil && blockNr == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "logsBloom", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

func blockByNumber(ctx context.Context, b Backend, blockNr rpc.BlockNumber) (*types.Block, error) {
	switch blockNr {
	case rpc.PendingBlockNumber:
		ps, ok := b.(PendingState)
		if !ok {
			return nil, fmt.Errorf("can't access the pending block with the current backend")
		}
		return ps.PendingBlock()
	case rpc.LatestBlockNumber:
		return b.BlockByNumber(ctx, nil)
	default:
		return b.BlockByNumber(ctx, big.NewInt(int64(blockNr)))
	}
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(ctx context.Context, blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := s.b.BlockByHash(ctx, blockHash)
	if block != nil {
		return s.rpcOutputBlock(ctx, block, true, fullTx)
	}
	return nil, err
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index rpc.HexNumber) (map[string]interface{}, error) {
	block, err := blockByNumber(ctx, s.b, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block #%d", index.Int(), blockNr)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(ctx, block, false, false)
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index rpc.HexNumber) (map[string]interface{}, error) {
	block, err := s.b.BlockByHash(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block %s", index.Int(), blockHash.Hex())
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(ctx, block, false, false)
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (s *PublicBlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) *rpc.HexNumber {
	if block, _ := blockByNumber(ctx, s.b, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) *rpc.HexNumber {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (s *PublicBlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*big.Int, error) {
	switch blockNr {
	case rpc.PendingBlockNumber:
		ps, ok := s.b.(PendingState)
		if !ok {
			return nil, ErrNoPendingState
		}
		return ps.PendingBalanceAt(ctx, address)
	case rpc.LatestBlockNumber:
		return s.b.BalanceAt(ctx, address, nil)
	default:
		return s.b.BalanceAt(ctx, address, big.NewInt(int64(blockNr)))
	}
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (string, error) {
	var code []byte
	var err error
	switch blockNr {
	case rpc.PendingBlockNumber:
		ps, ok := s.b.(PendingState)
		if !ok {
			return "", ErrNoPendingState
		}
		code, err = ps.PendingCodeAt(ctx, address)
	case rpc.LatestBlockNumber:
		code, err = s.b.CodeAt(ctx, address, nil)
	default:
		code, err = s.b.CodeAt(ctx, address, big.NewInt(int64(blockNr)))
	}

	if len(code) == 0 || err != nil { // backwards compatibility
		return "0x", err
	}
	return common.ToHex(code), nil
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *PublicBlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, key common.Hash, blockNr rpc.BlockNumber) (string, error) {
	var val []byte
	var err error
	switch blockNr {
	case rpc.PendingBlockNumber:
		ps, ok := s.b.(PendingState)
		if !ok {
			return "", ErrNoPendingState
		}
		val, err = ps.PendingStorageAt(ctx, address, key)
	case rpc.LatestBlockNumber:
		val, err = s.b.StorageAt(ctx, address, key, nil)
	default:
		val, err = s.b.StorageAt(ctx, address, key, big.NewInt(int64(blockNr)))
	}

	if len(val) == 0 || err != nil { // backwards compatibility
		return "0x", err
	}
	return common.ToHex(val), nil
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      rpc.HexNumber   `json:"gas"`
	GasPrice rpc.HexNumber   `json:"gasPrice"`
	Value    rpc.HexNumber   `json:"value"`
	Data     rpc.HexBytes    `json:"data"`
}

func (args CallArgs) Msg() ethereum.CallMsg {
	return ethereum.CallMsg{
		From:     args.From,
		To:       args.To,
		Gas:      args.Gas.BigInt(),
		GasPrice: args.GasPrice.BigInt(),
		Value:    args.Value.BigInt(),
		Data:     args.Data,
	}
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is usefull to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(ctx context.Context, args CallArgs, blockNr rpc.BlockNumber) (string, error) {
	var result []byte
	var err error
	switch blockNr {
	case rpc.PendingBlockNumber:
		ps, ok := s.b.(PendingState)
		if !ok {
			return "", ErrNoPendingState
		}
		result, err = ps.PendingCallContract(ctx, args.Msg())
	case rpc.LatestBlockNumber:
		result, err = s.b.CallContract(ctx, args.Msg(), nil)
	default:
		result, err = s.b.CallContract(ctx, args.Msg(), big.NewInt(int64(blockNr)))
	}
	return common.ToHex(result), err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (s *PublicBlockChainAPI) EstimateGas(ctx context.Context, args CallArgs) (*rpc.HexNumber, error) {
	gas, err := s.b.EstimateGas(ctx, args.Msg())
	return rpc.NewHexNumber(gas), err
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as the amount of
// gas used and the return value
type ExecutionResult struct {
	Gas         *big.Int       `json:"gas"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc      uint64            `json:"pc"`
	Op      string            `json:"op"`
	Gas     *big.Int          `json:"gas"`
	GasCost *big.Int          `json:"gasCost"`
	Depth   int               `json:"depth"`
	Error   error             `json:"error"`
	Stack   []string          `json:"stack"`
	Memory  []string          `json:"memory"`
	Storage map[string]string `json:"storage"`
}

// formatLogs formats EVM returned structured logs for json output
func FormatLogs(structLogs []vm.StructLog) []StructLogRes {
	formattedStructLogs := make([]StructLogRes, len(structLogs))
	for index, trace := range structLogs {
		formattedStructLogs[index] = StructLogRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   trace.Depth,
			Error:   trace.Err,
			Stack:   make([]string, len(trace.Stack)),
			Storage: make(map[string]string),
		}

		for i, stackValue := range trace.Stack {
			formattedStructLogs[index].Stack[i] = fmt.Sprintf("%x", common.LeftPadBytes(stackValue.Bytes(), 32))
		}

		for i := 0; i+32 <= len(trace.Memory); i += 32 {
			formattedStructLogs[index].Memory = append(formattedStructLogs[index].Memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
		}

		for i, storageValue := range trace.Storage {
			formattedStructLogs[index].Storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
		}
	}
	return formattedStructLogs
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainAPI) rpcOutputBlock(ctx context.Context, b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	head := b.Header() // copies the header once
	fields := map[string]interface{}{
		"number":           rpc.NewHexNumber(head.Number),
		"hash":             b.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       rpc.NewHexNumber(head.Difficulty),
		"totalDifficulty":  rpc.NewHexNumber(s.b.BlockTD(b.Hash())),
		"extraData":        rpc.HexBytes(head.Extra),
		"size":             rpc.NewHexNumber(b.Size().Int64()),
		"gasLimit":         rpc.NewHexNumber(head.GasLimit),
		"gasUsed":          rpc.NewHexNumber(head.GasUsed),
		"timestamp":        rpc.NewHexNumber(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}

	if inclTx {
		txs := b.Transactions()
		transactions := make([]interface{}, 0, len(txs))
		for i, tx := range b.Transactions() {
			if fullTx {
				rtx := newRPCTransaction(tx)
				rtx.setInclusionBlock(b.Hash(), b.NumberU64(), i)
				transactions = append(transactions, rtx)
			} else {
				transactions = append(transactions, tx.Hash())
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

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash     `json:"blockHash"`
	BlockNumber      *rpc.HexNumber  `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              *rpc.HexNumber  `json:"gas"`
	GasPrice         *rpc.HexNumber  `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            rpc.HexBytes    `json:"input"`
	Nonce            *rpc.HexNumber  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *rpc.HexNumber  `json:"transactionIndex"`
	Value            *rpc.HexNumber  `json:"value"`
	V                *rpc.HexNumber  `json:"v"`
	R                *rpc.HexNumber  `json:"r"`
	S                *rpc.HexNumber  `json:"s"`
}

func (tx *RPCTransaction) setInclusionBlock(blockhash common.Hash, blocknum uint64, index int) {
	tx.BlockHash = blockhash
	tx.BlockNumber = rpc.NewHexNumber(blocknum)
	tx.TransactionIndex = rpc.NewHexNumber(index)
}

// newRPCTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCTransaction(tx *types.Transaction) *RPCTransaction {
	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	return &RPCTransaction{
		From:     from,
		Gas:      rpc.NewHexNumber(tx.Gas()),
		GasPrice: rpc.NewHexNumber(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    rpc.HexBytes(tx.Data()),
		Nonce:    rpc.NewHexNumber(tx.Nonce()),
		To:       tx.To(),
		Value:    rpc.NewHexNumber(tx.Value()),
		V:        rpc.NewHexNumber(v),
		R:        rpc.NewHexNumber(r),
		S:        rpc.NewHexNumber(s),
	}
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, txIndex int) *RPCTransaction {
	if txIndex >= 0 && txIndex < len(b.Transactions()) {
		tx := newRPCTransaction(b.Transactions()[txIndex])
		tx.setInclusionBlock(b.Hash(), b.NumberU64(), txIndex)
		return tx
	}
	return nil
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *types.Block, txIndex int) (rpc.HexBytes, error) {
	if txIndex >= 0 && txIndex < len(b.Transactions()) {
		tx := b.Transactions()[txIndex]
		return rlp.EncodeToBytes(tx)
	}
	return nil, nil
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	b Backend
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(b Backend) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{b}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) *rpc.HexNumber {
	if block, _ := blockByNumber(ctx, s.b, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *rpc.HexNumber {
	count, err := s.b.TransactionCount(ctx, blockHash)
	if err == nil {
		return rpc.NewHexNumber(count)
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index rpc.HexNumber) *RPCTransaction {
	if block, _ := blockByNumber(ctx, s.b, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index rpc.HexNumber) *RPCTransaction {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil
}

// GetRawTransactionByBlockNumberAndIndex returns the bytes of the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index rpc.HexNumber) (rpc.HexBytes, error) {
	if block, _ := blockByNumber(ctx, s.b, blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetRawTransactionByBlockHashAndIndex returns the bytes of the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index rpc.HexNumber) (rpc.HexBytes, error) {
	if block, _ := s.b.BlockByHash(ctx, blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*rpc.HexNumber, error) {
	var nonce uint64
	var err error
	switch blockNr {
	case rpc.PendingBlockNumber:
		nonce, err = s.b.PendingNonceAt(ctx, address)
	case rpc.LatestBlockNumber:
		nonce, err = s.b.NonceAt(ctx, address, nil)
	default:
		nonce, err = s.b.NonceAt(ctx, address, big.NewInt(int64(blockNr)))
	}
	if err != nil {
		return nil, err
	}
	return rpc.NewHexNumber(nonce), nil
}

// getTransactionBlockData fetches the meta data for the given transaction from the chain database. This is useful to
// retrieve block information for a hash. It returns the block hash, block index and transaction index.
func getTransactionBlockData(chainDb ethdb.Database, txHash common.Hash) (common.Hash, uint64, uint64, error) {
	var txBlock struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}

	blockData, err := chainDb.Get(append(txHash.Bytes(), 0x0001))
	if err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	reader := bytes.NewReader(blockData)
	if err = rlp.Decode(reader, &txBlock); err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	return txBlock.BlockHash, txBlock.BlockIndex, txBlock.Index, nil
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolAPI) GetTransactionByHash(ctx context.Context, txhash common.Hash) (*RPCTransaction, error) {
	tx, isPending, err := s.b.TransactionByHash(ctx, txhash)
	if tx == nil || err != nil {
		return nil, nil
	}
	rtx := newRPCTransaction(tx)
	if isPending {
		return rtx, nil
	} else if tib, ok := s.b.(TransactionInclusionBlock); ok {
		bhash, bnum, index, err := tib.TransactionInclusionBlock(txhash)
		if err != nil {
			return nil, err
		}
		rtx.setInclusionBlock(bhash, bnum, index)
	}
	return rtx, nil
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (s *PublicTransactionPoolAPI) GetRawTransactionByHash(ctx context.Context, txhash common.Hash) (rpc.HexBytes, error) {
	tx, _, err := s.b.TransactionByHash(ctx, txhash)
	if tx == nil || err != nil {
		return nil, nil
	}
	return rlp.EncodeToBytes(tx)
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolAPI) GetTransactionReceipt(ctx context.Context, txhash common.Hash) (map[string]interface{}, error) {
	receipt, err := s.b.TransactionReceipt(ctx, txhash)
	if err != nil {
		glog.V(logger.Debug).Infof("can't find receipt for transaction %s: %v", txhash.Hex(), err)
		return nil, nil
	}
	tx, isPending, err := s.b.TransactionByHash(ctx, txhash)
	if err != nil {
		glog.V(logger.Debug).Infof("can't find transaction %s: %v\n", txhash.Hex(), err)
		return nil, nil
	}
	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, err := types.Sender(signer, tx)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"root":              rpc.HexBytes(receipt.PostState),
		"transactionHash":   txhash,
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           rpc.NewHexNumber(receipt.GasUsed),
		"cumulativeGasUsed": rpc.NewHexNumber(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
	}
	// Set block inclusion information if available.
	if tib, ok := s.b.(TransactionInclusionBlock); !isPending && ok {
		bhash, bnum, index, err := tib.TransactionInclusionBlock(txhash)
		if err != nil {
			glog.V(logger.Debug).Infof("%v\n", err)
			return nil, nil
		}
		fields["blockHash"] = bhash
		fields["blockNumber"] = rpc.NewHexNumber(bnum)
		fields["transactionIndex"] = rpc.NewHexNumber(index)
	}

	if receipt.Logs == nil {
		fields["logs"] = []vm.Logs{}
	}
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolAPI) sign(ctx context.Context, addr common.Address, tx *types.Transaction) ([]byte, types.Signer, error) {
	head, err := s.b.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	signer := types.MakeSigner(s.b.ChainConfig(), head.Number)
	sig, err := s.b.AccountManager().SignEthereum(addr, signer.Hash(tx).Bytes())
	return sig, signer, err
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
}

// prepareSendTxArgs is a helper function that fills in default values for unspecified tx fields.
func prepareSendTxArgs(ctx context.Context, args SendTxArgs, b Backend) (SendTxArgs, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		price, err := b.SuggestGasPrice(ctx)
		if err != nil {
			return args, err
		}
		args.GasPrice = rpc.NewHexNumber(price)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}
	return args, nil
}

// submitTransaction is a helper function that submits tx to txPool and creates a log entry.
func submitTransaction(ctx context.Context, b Backend, tx *types.Transaction, signer types.Signer, signature []byte) (common.Hash, error) {
	signedTx := signer.WithSignature(tx, signature)
	if err := b.SendTransaction(ctx, signedTx); err != nil {
		return common.Hash{}, err
	}
	if signedTx.To() == nil {
		from, _ := types.Sender(signer, signedTx)
		addr := crypto.CreateAddress(from, signedTx.Nonce())
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signedTx.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signedTx.Hash().Hex(), tx.To().Hex())
	}

	return signedTx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolAPI) SendTransaction(ctx context.Context, args SendTxArgs) (common.Hash, error) {
	var err error
	args, err = prepareSendTxArgs(ctx, args, s.b)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		nonce, err := s.b.PendingNonceAt(ctx, args.From)
		if err != nil {
			return common.Hash{}, err
		}
		args.Nonce = rpc.NewHexNumber(nonce)
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	head, err := s.b.HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, err
	}
	signer := types.MakeSigner(s.b.ChainConfig(), head.Number)
	signature, err := s.b.AccountManager().SignEthereum(args.From, signer.Hash(tx).Bytes())
	if err != nil {
		return common.Hash{}, err
	}

	return submitTransaction(ctx, s.b, tx, signer, signature)
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransaction(ctx context.Context, encodedTx string) (string, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(common.FromHex(encodedTx), tx); err != nil {
		return "", err
	}
	if err := s.b.SendTransaction(ctx, tx); err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

// Sign calculates an ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message).
//
// The account associated with addr must be unlocked.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
func (s *PublicTransactionPoolAPI) Sign(addr common.Address, message string) (string, error) {
	hash := signHash(message)
	signature, err := s.b.AccountManager().SignEthereum(addr, hash)
	return common.ToHex(signature), err
}

// SignTransactionArgs represents the arguments to sign a transaction.
type SignTransactionArgs struct {
	From     common.Address
	To       *common.Address
	Nonce    *rpc.HexNumber
	Value    *rpc.HexNumber
	Gas      *rpc.HexNumber
	GasPrice *rpc.HexNumber
	Data     string

	BlockNumber int64
}

// Tx is a helper object for argument and return values
type Tx struct {
	tx *types.Transaction

	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	GasLimit *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Hash     common.Hash     `json:"hash"`
}

// UnmarshalJSON parses JSON data into tx.
func (tx *Tx) UnmarshalJSON(b []byte) (err error) {
	req := struct {
		To       *common.Address `json:"to"`
		From     common.Address  `json:"from"`
		Nonce    *rpc.HexNumber  `json:"nonce"`
		Value    *rpc.HexNumber  `json:"value"`
		Data     string          `json:"data"`
		GasLimit *rpc.HexNumber  `json:"gas"`
		GasPrice *rpc.HexNumber  `json:"gasPrice"`
		Hash     common.Hash     `json:"hash"`
	}{}

	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}

	tx.To = req.To
	tx.From = req.From
	tx.Nonce = req.Nonce
	tx.Value = req.Value
	tx.Data = req.Data
	tx.GasLimit = req.GasLimit
	tx.GasPrice = req.GasPrice
	tx.Hash = req.Hash

	data := common.Hex2Bytes(tx.Data)

	if tx.Nonce == nil {
		return fmt.Errorf("need nonce")
	}
	if tx.Value == nil {
		tx.Value = rpc.NewHexNumber(0)
	}
	if tx.GasLimit == nil {
		tx.GasLimit = rpc.NewHexNumber(0)
	}
	if tx.GasPrice == nil {
		tx.GasPrice = rpc.NewHexNumber(int64(50000000000))
	}

	if req.To == nil {
		tx.tx = types.NewContractCreation(tx.Nonce.Uint64(), tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	} else {
		tx.tx = types.NewTransaction(tx.Nonce.Uint64(), *tx.To, tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	}

	return nil
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw string `json:"raw"`
	Tx  *Tx    `json:"tx"`
}

func newTx(t *types.Transaction) *Tx {
	var signer types.Signer = types.HomesteadSigner{}
	if t.Protected() {
		signer = types.NewEIP155Signer(t.ChainId())
	}

	from, _ := types.Sender(signer, t)
	return &Tx{
		tx:       t,
		To:       t.To(),
		From:     from,
		Value:    rpc.NewHexNumber(t.Value()),
		Nonce:    rpc.NewHexNumber(t.Nonce()),
		Data:     "0x" + common.Bytes2Hex(t.Data()),
		GasLimit: rpc.NewHexNumber(t.Gas()),
		GasPrice: rpc.NewHexNumber(t.GasPrice()),
		Hash:     t.Hash(),
	}
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolAPI) SignTransaction(ctx context.Context, args SignTransactionArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		price, err := s.b.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
		args.GasPrice = rpc.NewHexNumber(price)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	if args.Nonce == nil {
		nonce, err := s.b.PendingNonceAt(ctx, args.From)
		if err != nil {
			return nil, err
		}
		args.Nonce = rpc.NewHexNumber(nonce)
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signature, signer, err := s.sign(ctx, args.From, tx)
	if err != nil {
		return nil, err
	}
	signedTx := signer.WithSignature(tx, signature)
	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}

	return &SignTransactionResult{"0x" + common.Bytes2Hex(data), newTx(signedTx)}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool and have a from address that is one of
// the accounts this node manages.
func (s *PublicTransactionPoolAPI) PendingTransactions() []*RPCTransaction {
	pending := s.b.PendingTransactions()
	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if tx.Protected() {
			signer = types.NewEIP155Signer(tx.ChainId())
		}
		from, _ := types.Sender(signer, tx)
		if s.b.AccountManager().HasAddress(from) {
			transactions = append(transactions, newRPCTransaction(tx))
		}
	}
	return transactions
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove the given transaction from the
// pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolAPI) Resend(ctx context.Context, tx Tx, gasPrice, gasLimit *rpc.HexNumber) (common.Hash, error) {
	pending := s.b.PendingTransactions()
	for _, p := range pending {
		var signer types.Signer = types.HomesteadSigner{}
		if p.Protected() {
			signer = types.NewEIP155Signer(p.ChainId())
		}

		if pFrom, err := types.Sender(signer, p); err == nil && pFrom == tx.From && signer.Hash(p) == signer.Hash(tx.tx) {
			if gasPrice == nil {
				gasPrice = rpc.NewHexNumber(tx.tx.GasPrice())
			}
			if gasLimit == nil {
				gasLimit = rpc.NewHexNumber(tx.tx.Gas())
			}

			var newTx *types.Transaction
			if tx.tx.To() == nil {
				newTx = types.NewContractCreation(tx.tx.Nonce(), tx.tx.Value(), gasLimit.BigInt(), gasPrice.BigInt(), tx.tx.Data())
			} else {
				newTx = types.NewTransaction(tx.tx.Nonce(), *tx.tx.To(), tx.tx.Value(), gasLimit.BigInt(), gasPrice.BigInt(), tx.tx.Data())
			}

			signature, signer, err := s.sign(ctx, tx.From, newTx)
			if err != nil {
				return common.Hash{}, err
			}
			newTx = signer.WithSignature(newTx, signature)

			s.b.RemoveTransaction(tx.Hash)
			return submitTransaction(ctx, s.b, newTx, signer, signature)
		}
	}

	return common.Hash{}, fmt.Errorf("Transaction %#x not found", tx.Hash)
}

// PublicDebugAPI is the collection of Etheruem APIs exposed over the public
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
func (api *PublicDebugAPI) GetBlockRlp(ctx context.Context, number rpc.BlockNumber) (string, error) {
	block, _ := blockByNumber(ctx, api.b, number)
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
func (api *PublicDebugAPI) PrintBlock(ctx context.Context, number rpc.BlockNumber) (string, error) {
	block, _ := blockByNumber(ctx, api.b, number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return fmt.Sprintf("%s", block), nil
}

// SeedHash retrieves the seed hash of a block.
func (api *PublicDebugAPI) SeedHash(ctx context.Context, number uint64) (string, error) {
	hash, err := ethash.GetSeedHash(number)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", hash), nil
}

// PrivateDebugAPI is the collection of Ethereum APIs exposed over the private
// debugging endpoint.
type PrivateDebugAPI struct {
	b       Backend
	chaindb ethdb.Database
}

// NewPrivateDebugAPI creates a new API definition for the private debug methods
// of the Ethereum service.
func NewPrivateDebugAPI(b Backend, chaindb ethdb.Database) *PrivateDebugAPI {
	return &PrivateDebugAPI{b, chaindb}
}

// ChaindbProperty returns leveldb properties of the chain database.
func (api *PrivateDebugAPI) ChaindbProperty(property string) (string, error) {
	ldb, ok := api.chaindb.(interface {
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
	ldb, ok := api.chaindb.(interface {
		LDB() *leveldb.DB
	})
	if !ok {
		return fmt.Errorf("chaindbCompact does not work for memory databases")
	}
	for b := byte(0); b < 255; b++ {
		glog.V(logger.Info).Infof("compacting chain DB range 0x%0.2X-0x%0.2X", b, b+1)
		err := ldb.LDB().CompactRange(util.Range{Start: []byte{b}, Limit: []byte{b + 1}})
		if err != nil {
			glog.Errorf("compaction error: %v", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *PrivateDebugAPI) SetHead(number rpc.HexNumber) {
	api.b.ResetHeadBlock(number.Uint64())
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion int
}

// NewPublicNetAPI creates a new net API instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion int) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
