package core

import (
	"fmt"
	"math/big"

	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

const (
	FilterTimeout = 300 * time.Second // Remove filter after FilterTimeout
)

type BlockChainService struct {
	bc *BlockChain
	am *accounts.Manager
}

func NewBlockChainService(bc *BlockChain, am *accounts.Manager) *BlockChainService {
	return &BlockChainService{bc: bc, am: am}
}

// BlockNumber returns the block number of the chain head.
func (s *BlockChainService) BlockNumber() *big.Int {
	return s.bc.CurrentHeader().Number
}

// GetBalance returns the amount of wei for the given address in the state of the given block number.
// When block number equals rpc.LatestBlockNumber the current block is used.
func (s *BlockChainService) GetBalance(address common.Address, blockNr rpc.BlockNumber) (*big.Int, error) {
	block := blockByNumber(s.bc, blockNr)
	if block == nil {
		return nil, nil
	}

	state, err := state.New(block.Root(), s.bc.chainDb)
	if err != nil {
		return nil, err
	}
	return state.GetBalance(address), nil
}

// blockByNumber is a commonly used helper function which retrieves and returns the block for the given block number. It
// returns nil when no block could be found.
func blockByNumber(bc *BlockChain, blockNr rpc.BlockNumber) *types.Block {
	if blockNr == rpc.LatestBlockNumber {
		return bc.CurrentBlock()
	}

	return bc.GetBlockByNumber(uint64(blockNr))
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *BlockChainService) GetBlockByNumber(blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *BlockChainService) GetBlockByHash(blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *BlockChainService) GetUncleByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (map[string]interface{}, error) {
	if blockNr == rpc.PendingBlockNumber {
		return nil, nil
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block #%d", index.Int(), blockNr)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, nil
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *BlockChainService) GetUncleByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block %s", index.Int(), blockHash.Hex())
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, nil
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (s *BlockChainService) GetUncleCountByBlockNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *BlockChainService) GetUncleCountByBlockHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// NewBlocksArgs allows the user to specify if the returned block should include transactions and in which format.
type NewBlocksArgs struct {
	IncludeTransactions bool `json:"includeTransactions"`
	TransactionDetails  bool `json:"transactionDetails"`
}

// NewBlocks triggers a new block event each time a block is appended to the chain. It accepts an argument which allows
// the caller to specify whether the output should contain transactions and in what format.
func (s *BlockChainService) NewBlocks(args NewBlocksArgs) (rpc.Subscription, error) {
	sub := s.bc.eventMux.Subscribe(ChainEvent{})

	output := func(rawBlock interface{}) interface{} {
		if event, ok := rawBlock.(ChainEvent); ok {
			notification, err := s.rpcOutputBlock(event.Block, args.IncludeTransactions, args.TransactionDetails)
			if err == nil {
				return notification
			}
		}
		return rawBlock
	}

	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}

func (s *BlockChainService) GetCode(address common.Address, blockNr rpc.BlockNumber) (string, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		state, err := state.New(block.Root(), s.bc.chainDb)
		if err != nil {
			return "", err
		}
		res := state.GetCode(address)
		if len(res) == 0 { // backwards compatibility
			return "0x", nil
		}
		return common.ToHex(res), nil
	}

	return "0x", nil
}

// StorageAt returns the storage at a given address
func (s *BlockChainService) GetStorageAt(address common.Address, position string, blockNr rpc.BlockNumber) (string, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		state, err := state.New(block.Root(), s.bc.chainDb)
		if err != nil {
			return "", err
		}
		return state.GetState(address, common.HexToHash(position)).Hex(), nil
	}

	return "0x", nil
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64 { return m.from.Nonce() }
func (m callmsg) To() *common.Address { return m.to }
func (m callmsg) GasPrice() *big.Int { return m.gasPrice }
func (m callmsg) Gas() *big.Int { return m.gas }
func (m callmsg) Value() *big.Int { return m.value }
func (m callmsg) Data() []byte { return m.data }

type CallArgs struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      rpc.HexNumber  `json:"gas"`
	GasPrice rpc.HexNumber  `json:"gasPrice"`
	Value    rpc.HexNumber  `json:"value"`
	Data     string         `json:"data"`
}

func (s *BlockChainService) doCall(args CallArgs, blockNr rpc.BlockNumber) (string, *big.Int, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		stateDb, err := state.New(block.Root(), s.bc.chainDb)
		if err != nil {
			return "0x", nil, err
		}

		stateDb = stateDb.Copy()
		var from *state.StateObject
		if args.From == (common.Address{}) {
			accounts, err := s.am.Accounts()
			if err != nil || len(accounts) == 0 {
				from = stateDb.GetOrNewStateObject(common.Address{})
			} else {
				from = stateDb.GetOrNewStateObject(accounts[0].Address)
			}
		} else {
			from = stateDb.GetOrNewStateObject(args.From)
		}

		from.SetBalance(common.MaxBig)

		msg := callmsg{
			from:     from,
			to:       &args.To,
			gas:      args.Gas.BigInt(),
			gasPrice: args.GasPrice.BigInt(),
			value:    args.Value.BigInt(),
			data:     common.FromHex(args.Data),
		}

		if msg.gas.Cmp(common.Big0) == 0 {
			msg.gas = big.NewInt(50000000)
		}

		if msg.gasPrice.Cmp(common.Big0) == 0 {
			msg.gasPrice = new(big.Int).Mul(big.NewInt(50), common.Shannon)
		}

		header := s.bc.CurrentBlock().Header()
		vmenv := NewEnv(stateDb, s.bc, msg, header)
		gp := new(GasPool).AddGas(common.MaxBig)
		res, gas, err := ApplyMessage(vmenv, msg, gp)
		if len(res) == 0 { // backwards compatability
			return "0x", gas, err
		}
		return common.ToHex(res), gas, err
	}

	return "0x", common.Big0, nil
}

func (s *BlockChainService) Call(args CallArgs, blockNr rpc.BlockNumber) (string, error) {
	result, _, err := s.doCall(args, blockNr)
	return result, err
}

func (s *BlockChainService) EstimateGas(args CallArgs) (*rpc.HexNumber, error) {
	_, gas, err := s.doCall(args, rpc.LatestBlockNumber)
	return rpc.NewHexNumber(gas), err
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *BlockChainService) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields := map[string]interface{}{
		"number":           rpc.NewHexNumber(b.Number()),
		"hash":             b.Hash(),
		"parentHash":       b.ParentHash(),
		"nonce":            b.Header().Nonce,
		"sha3Uncles":       b.UncleHash(),
		"logsBloom":        b.Bloom(),
		"stateRoot":        b.Root(),
		"miner":            b.Coinbase(),
		"difficulty":       rpc.NewHexNumber(b.Difficulty()),
		"totalDifficulty":  rpc.NewHexNumber(s.bc.GetTd(b.Hash())),
		"extraData":        fmt.Sprintf("0x%x", b.Extra()),
		"size":             rpc.NewHexNumber(b.Size().Int64()),
		"gasLimit":         rpc.NewHexNumber(b.GasLimit()),
		"gasUsed":          rpc.NewHexNumber(b.GasUsed()),
		"timestamp":        rpc.NewHexNumber(b.Time()),
		"transactionsRoot": b.TxHash(),
		"receiptRoot":      b.ReceiptHash(),
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}

		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransaction(b, tx.Hash())
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