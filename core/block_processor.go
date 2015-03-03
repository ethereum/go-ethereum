package core

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/pow/ezp"
	"github.com/ethereum/go-ethereum/state"
	"gopkg.in/fatih/set.v0"
)

type PendingBlockEvent struct {
	Block *types.Block
}

var statelogger = logger.NewLogger("BLOCK")

type BlockProcessor struct {
	db ethutil.Database
	// Mutex for locking the block processor. Blocks can only be handled one at a time
	mutex sync.Mutex
	// Canonical block chain
	bc *ChainManager
	// non-persistent key/value memory storage
	mem map[string]*big.Int
	// Proof of work used for validating
	Pow pow.PoW

	txpool *TxPool

	// The last attempted block is mainly used for debugging purposes
	// This does not have to be a valid block and will be set during
	// 'Process' & canonical validation.
	lastAttemptedBlock *types.Block

	events event.Subscription

	eventMux *event.TypeMux
}

func NewBlockProcessor(db ethutil.Database, txpool *TxPool, chainManager *ChainManager, eventMux *event.TypeMux) *BlockProcessor {
	sm := &BlockProcessor{
		db:       db,
		mem:      make(map[string]*big.Int),
		Pow:      ezp.New(),
		bc:       chainManager,
		eventMux: eventMux,
		txpool:   txpool,
	}

	return sm
}

func (sm *BlockProcessor) TransitionState(statedb *state.StateDB, parent, block *types.Block, transientProcess bool) (receipts types.Receipts, err error) {
	coinbase := statedb.GetOrNewStateObject(block.Header().Coinbase)
	coinbase.SetGasPool(block.Header().GasLimit)

	// Process the transactions on to parent state
	receipts, _, _, _, err = sm.ApplyTransactions(coinbase, statedb, block, block.Transactions(), transientProcess)
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (self *BlockProcessor) ApplyTransaction(coinbase *state.StateObject, statedb *state.StateDB, block *types.Block, tx *types.Transaction, usedGas *big.Int, transientProcess bool) (*types.Receipt, *big.Int, error) {
	// If we are mining this block and validating we want to set the logs back to 0
	statedb.EmptyLogs()

	txGas := new(big.Int).Set(tx.Gas())

	cb := statedb.GetStateObject(coinbase.Address())
	st := NewStateTransition(NewEnv(statedb, self.bc, tx, block), tx, cb)
	_, err := st.TransitionState()
	if err != nil && (IsNonceErr(err) || state.IsGasLimitErr(err)) {
		return nil, nil, err
	}

	txGas.Sub(txGas, st.gas)

	// Update the state with pending changes
	statedb.Update(txGas)

	cumulative := new(big.Int).Set(usedGas.Add(usedGas, txGas))
	receipt := types.NewReceipt(statedb.Root(), cumulative)
	receipt.SetLogs(statedb.Logs())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	chainlogger.Debugln(receipt)

	// Notify all subscribers
	if !transientProcess {
		go self.eventMux.Post(TxPostEvent{tx})
		logs := statedb.Logs()
		go self.eventMux.Post(logs)
	}

	return receipt, txGas, err
}

func (self *BlockProcessor) ApplyTransactions(coinbase *state.StateObject, statedb *state.StateDB, block *types.Block, txs types.Transactions, transientProcess bool) (types.Receipts, types.Transactions, types.Transactions, types.Transactions, error) {
	var (
		receipts           types.Receipts
		handled, unhandled types.Transactions
		erroneous          types.Transactions
		totalUsedGas       = big.NewInt(0)
		err                error
		cumulativeSum      = new(big.Int)
	)

	for _, tx := range txs {
		receipt, txGas, err := self.ApplyTransaction(coinbase, statedb, block, tx, totalUsedGas, transientProcess)
		if err != nil {
			switch {
			case IsNonceErr(err):
				return nil, nil, nil, nil, err
			case state.IsGasLimitErr(err):
				return nil, nil, nil, nil, err
			default:
				statelogger.Infoln(err)
				erroneous = append(erroneous, tx)
				err = nil
			}
		}
		receipts = append(receipts, receipt)
		handled = append(handled, tx)

		cumulativeSum.Add(cumulativeSum, new(big.Int).Mul(txGas, tx.GasPrice()))
	}

	block.Reward = cumulativeSum
	block.Header().GasUsed = totalUsedGas

	if transientProcess {
		go self.eventMux.Post(PendingBlockEvent{block})
	}

	return receipts, handled, unhandled, erroneous, err
}

// Process block will attempt to process the given block's transactions and applies them
// on top of the block's parent state (given it exists) and will return wether it was
// successful or not.
func (sm *BlockProcessor) Process(block *types.Block) (td *big.Int, err error) {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	header := block.Header()
	if sm.bc.HasBlock(header.Hash()) {
		return nil, &KnownBlockError{header.Number, header.Hash()}
	}

	if !sm.bc.HasBlock(header.ParentHash) {
		return nil, ParentError(header.ParentHash)
	}
	parent := sm.bc.GetBlock(header.ParentHash)

	return sm.processWithParent(block, parent)
}

func (sm *BlockProcessor) processWithParent(block, parent *types.Block) (td *big.Int, err error) {
	sm.lastAttemptedBlock = block

	// Create a new state based on the parent's root (e.g., create copy)
	state := state.New(parent.Root(), sm.db)

	// Block validation
	if err = sm.ValidateBlock(block, parent); err != nil {
		return
	}

	receipts, err := sm.TransitionState(state, parent, block, false)
	if err != nil {
		return
	}

	header := block.Header()

	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if bytes.Compare(rbloom, header.Bloom) != 0 {
		err = fmt.Errorf("unable to replicate block's bloom=%x", rbloom)
		return
	}

	// The transactions Trie's root (R = (Tr [[H1, T1], [H2, T2], ... [Hn, Tn]]))
	// can be used by light clients to make sure they've received the correct Txs
	txSha := types.DeriveSha(block.Transactions())
	if bytes.Compare(txSha, header.TxHash) != 0 {
		err = fmt.Errorf("validating transaction root. received=%x got=%x", header.TxHash, txSha)
		return
	}

	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveSha(receipts)
	if bytes.Compare(receiptSha, header.ReceiptHash) != 0 {
		err = fmt.Errorf("validating receipt root. received=%x got=%x", header.ReceiptHash, receiptSha)
		return
	}

	// Accumulate static rewards; block reward, uncle's and uncle inclusion.
	if err = sm.AccumulateRewards(state, block, parent); err != nil {
		return
	}

	// Commit state objects/accounts to a temporary trie (does not save)
	// used to calculate the state root.
	state.Update(ethutil.Big0)
	if !bytes.Equal(header.Root, state.Root()) {
		err = fmt.Errorf("invalid merkle root. received=%x got=%x", header.Root, state.Root())
		return
	}

	// Calculate the td for this block
	td = CalculateTD(block, parent)
	// Sync the current block's state to the database
	state.Sync()
	// Remove transactions from the pool
	sm.txpool.RemoveSet(block.Transactions())

	chainlogger.Infof("processed block #%d (%x...)\n", header.Number, block.Hash()[0:4])

	return td, nil
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *BlockProcessor) ValidateBlock(block, parent *types.Block) error {
	if len(block.Header().Extra) > 1024 {
		return fmt.Errorf("Block extra data too long (%d)", len(block.Header().Extra))
	}

	expd := CalcDifficulty(block, parent)
	if expd.Cmp(block.Header().Difficulty) != 0 {
		return fmt.Errorf("Difficulty check failed for block %v, %v", block.Header().Difficulty, expd)
	}

	//expl := CalcGasLimit(parent, block)
	//if expl.Cmp(block.Header().GasLimit) != 0 {

	// block.gasLimit - parent.gasLimit <= parent.gasLimit / 1024
	a := new(big.Int).Sub(block.Header().GasLimit, parent.Header().GasLimit)
	b := new(big.Int).Div(parent.Header().GasLimit, big.NewInt(1024))
	if a.Cmp(b) > 0 {
		return fmt.Errorf("GasLimit check failed for block %v", block.Header().GasLimit)
	}

	if block.Time() < parent.Time() {
		return ValidationError("Block timestamp not after prev block (%v - %v)", block.Header().Time, parent.Header().Time)
	}

	if block.Time() > time.Now().Unix() {
		return BlockFutureErr
	}

	if new(big.Int).Sub(block.Number(), parent.Number()).Cmp(big.NewInt(1)) != 0 {
		return BlockNumberErr
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !sm.Pow.Verify(block) {
		return ValidationError("Block's nonce is invalid (= %v)", ethutil.Bytes2Hex(block.Header().Nonce))
	}

	return nil
}

func (sm *BlockProcessor) AccumulateRewards(statedb *state.StateDB, block, parent *types.Block) error {
	reward := new(big.Int).Set(BlockReward)

	ancestors := set.New()
	for _, ancestor := range sm.bc.GetAncestors(block, 7) {
		ancestors.Add(string(ancestor.Hash()))
	}

	uncles := set.New()
	uncles.Add(string(block.Hash()))
	for _, uncle := range block.Uncles() {
		if uncles.Has(string(uncle.Hash())) {
			// Error not unique
			return UncleError("Uncle not unique")
		}
		uncles.Add(string(uncle.Hash()))

		if !ancestors.Has(string(uncle.ParentHash)) {
			return UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
		}

		if !sm.Pow.Verify(types.NewBlockWithHeader(uncle)) {
			return ValidationError("Uncle's nonce is invalid (= %v)", ethutil.Bytes2Hex(uncle.Nonce))
		}

		r := new(big.Int)
		r.Mul(BlockReward, big.NewInt(15)).Div(r, big.NewInt(16))

		statedb.AddBalance(uncle.Coinbase, r)

		reward.Add(reward, new(big.Int).Div(BlockReward, big.NewInt(32)))
	}

	// Get the account associated with the coinbase
	statedb.AddBalance(block.Header().Coinbase, reward)

	return nil
}

func (sm *BlockProcessor) GetLogs(block *types.Block) (logs state.Logs, err error) {
	if !sm.bc.HasBlock(block.Header().ParentHash) {
		return nil, ParentError(block.Header().ParentHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.Header().ParentHash)
		state  = state.New(parent.Root(), sm.db)
	)

	sm.TransitionState(state, parent, block, true)
	sm.AccumulateRewards(state, block, parent)

	return state.Logs(), nil
}
