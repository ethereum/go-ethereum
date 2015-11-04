package core

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"sync"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/event"
)

const (
	defaultGasPrice = uint64(10000000000000)
	defaultGas = uint64(90000)
)

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash     `json:"blockHash"`
	BlockNumber      *rpc.HexNumber  `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              *rpc.HexNumber  `json:"gas"`
	GasPrice         *rpc.HexNumber  `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            string          `json:"input"`
	Nonce            *rpc.HexNumber  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *rpc.HexNumber  `json:"transactionIndex"`
	Value            *rpc.HexNumber  `json:"value"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, txIndex int) (*RPCTransaction, error) {
	if txIndex >= 0 && txIndex < len(b.Transactions()) {
		tx := b.Transactions()[txIndex]
		from, err := tx.From()
		if err != nil {
			return nil, err
		}

		return &RPCTransaction{
			BlockHash:        b.Hash(),
			BlockNumber:      rpc.NewHexNumber(b.Number()),
			From:             from,
			Gas:              rpc.NewHexNumber(tx.Gas()),
			GasPrice:         rpc.NewHexNumber(tx.GasPrice()),
			Hash:             tx.Hash(),
			Input:            fmt.Sprintf("0x%x", tx.Data()),
			Nonce:            rpc.NewHexNumber(tx.Nonce()),
			To:               tx.To(),
			TransactionIndex: rpc.NewHexNumber(txIndex),
			Value:            rpc.NewHexNumber(tx.Value()),
		}, nil
	}

	return nil, nil
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransaction(b *types.Block, txHash common.Hash) (*RPCTransaction, error) {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == txHash {
			return newRPCTransactionFromBlockIndex(b, idx)
		}
	}

	return nil, nil
}

// TransactionPoolService exposes methods for the RPC interface
type TransactionPoolService struct {
	eventMux *event.TypeMux
	chainDb  ethdb.Database
	bc       *BlockChain
	am       *accounts.Manager
	txPool   *TxPool
	txMu     sync.Mutex
}

// NewTransactionPoolService creates a new RPC service with methods specific for the transaction pool.
func NewTransactionPoolService(txPool *TxPool, chainDb ethdb.Database, bc *BlockChain, am *accounts.Manager) *TransactionPoolService {
	return &TransactionPoolService{
		eventMux: txPool.eventMux,
		chainDb: chainDb,
		bc: bc,
		am: am,
		txPool: txPool,
	}
}

func getTransaction(chainDb ethdb.Database, txPool *TxPool, txHash common.Hash) (*types.Transaction, error) {
	txData, err := chainDb.Get(txHash.Bytes())
	if err != nil {
		return nil, err
	}

	tx := new(types.Transaction)
	if len(txData) > 0 {
		if err := rlp.DecodeBytes(txData, tx); err != nil {
			return nil, err
		}
	} else { // pending transaction?
		tx = txPool.GetTransaction(txHash)
	}

	return tx, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *TransactionPoolService) GetBlockTransactionCountByNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}

	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *TransactionPoolService) GetBlockTransactionCountByHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *TransactionPoolService) GetTransactionByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *TransactionPoolService) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *TransactionPoolService) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (*rpc.HexNumber, error) {
	block := blockByNumber(s.bc, blockNr)
	if block == nil {
		return nil, nil
	}

	state, err := state.New(block.Root(), s.chainDb)
	if err != nil {
		return nil, err
	}
	return rpc.NewHexNumber(state.GetNonce(address)), nil
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
func (s *TransactionPoolService) GetTransactionByHash(txHash common.Hash) (*RPCTransaction, error) {
	var tx *types.Transaction
	var err error

	if tx, err = getTransaction(s.chainDb, s.txPool, txHash); err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	} else if tx == nil {
		return nil, nil
	}

	blockHash, _, _, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransaction(block, txHash)
	}

	return nil, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *TransactionPoolService) GetTransactionReceipt(txHash common.Hash) (map[string]interface{}, error) {
	receipt := GetReceipt(s.chainDb, txHash)
	if receipt == nil {
		glog.V(logger.Debug).Infof("receipt not found for transaction %s", txHash.Hex())
		return nil, nil
	}

	tx, err := getTransaction(s.chainDb, s.txPool, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	txBlock, blockIndex, index, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	from, err := tx.From()
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	fields := map[string]interface{}{
		"blockHash":         txBlock,
		"blockNumber":       rpc.NewHexNumber(blockIndex),
		"transactionHash":   txHash,
		"transactionIndex":  rpc.NewHexNumber(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           rpc.NewHexNumber(receipt.GasUsed),
		"cumulativeGasUsed": rpc.NewHexNumber(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
	}

	if receipt.Logs == nil {
		fields["logs"] = []vm.Logs{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if bytes.Compare(receipt.ContractAddress.Bytes(), bytes.Repeat([]byte{0}, 20)) != 0 {
		fields["contractAddress"] = receipt.ContractAddress
	}

	return fields, nil
}


func (s *TransactionPoolService) sign(from common.Address, tx *types.Transaction) (*types.Transaction, error) {
	acc := accounts.Account{from}
	signature, err := s.am.Sign(acc, tx.SigHash().Bytes())
	if err != nil {
		return nil, err
	}

	return tx.WithSignature(signature)
}

type SendTxArgs struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      *rpc.HexNumber `json:"gas"`
	GasPrice *rpc.HexNumber `json:"gasPrice"`
	Value    *rpc.HexNumber `json:"value"`
	Data     string         `json:"data"`
	Nonce    *rpc.HexNumber `json:"nonce"`
}

func (s *TransactionPoolService) SendTransaction(args SendTxArgs) (common.Hash, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	contractCreation := (args.To == common.Address{})

	if contractCreation {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signedTx, err := s.sign(args.From, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if err := s.txPool.Add(signedTx); err != nil {
		return common.Hash{}, nil
	}

	if contractCreation {
		addr := crypto.CreateAddress(args.From, args.Nonce.Uint64())
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signedTx.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signedTx.Hash().Hex(), tx.To().Hex())
	}

	return signedTx.Hash(), nil
}

func (s *TransactionPoolService) PendingTransactions() (rpc.Subscription, error) {
	sub := s.eventMux.Subscribe(TxPreEvent{})

	output := func(transaction interface{}) interface{} {
		if tx, ok := transaction.(TxPreEvent); ok {
			return tx.Tx.Hash()
		}
		return nil
	}

	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}
