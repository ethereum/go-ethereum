package core

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/state"
	"gopkg.in/fatih/set.v0"
)

type PendingBlockEvent struct {
	Block *types.Block
}

var statelogger = logger.NewLogger("BLOCK")

type BlockProcessor struct {
	db      common.Database
	extraDb common.Database
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

func NewBlockProcessor(db, extra common.Database, pow pow.PoW, txpool *TxPool, chainManager *ChainManager, eventMux *event.TypeMux) *BlockProcessor {
	sm := &BlockProcessor{
		db:       db,
		extraDb:  extra,
		mem:      make(map[string]*big.Int),
		Pow:      pow,
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

	cb := statedb.GetStateObject(coinbase.Address())
	_, gas, err := ApplyMessage(NewEnv(statedb, self.bc, tx, block), tx, cb)
	if err != nil && (IsNonceErr(err) || state.IsGasLimitErr(err) || IsInvalidTxErr(err)) {
		// If the account is managed, remove the invalid nonce.
		self.bc.TxState().RemoveNonce(tx.From(), tx.Nonce())
		return nil, nil, err
	}

	// Update the state with pending changes
	statedb.Update(nil)

	cumulative := new(big.Int).Set(usedGas.Add(usedGas, gas))
	receipt := types.NewReceipt(statedb.Root().Bytes(), cumulative)
	receipt.SetLogs(statedb.Logs())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	chainlogger.Debugln(receipt)

	// Notify all subscribers
	if !transientProcess {
		go self.eventMux.Post(TxPostEvent{tx})
		logs := statedb.Logs()
		go self.eventMux.Post(logs)
	}

	return receipt, gas, err
}
func (self *BlockProcessor) ChainManager() *ChainManager {
	return self.bc
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
		if err != nil && (IsNonceErr(err) || state.IsGasLimitErr(err) || IsInvalidTxErr(err)) {
			return nil, nil, nil, nil, err
		}

		if err != nil {
			statelogger.Infoln("TX err:", err)
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
	if err = sm.ValidateHeader(block.Header(), parent.Header()); err != nil {
		return
	}

	// There can be at most two uncles
	if len(block.Uncles()) > 2 {
		return nil, ValidationError("Block can only contain one uncle (contained %v)", len(block.Uncles()))
	}

	receipts, err := sm.TransitionState(state, parent, block, false)
	if err != nil {
		return
	}

	header := block.Header()

	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		err = fmt.Errorf("unable to replicate block's bloom=%x", rbloom)
		return
	}

	// The transactions Trie's root (R = (Tr [[H1, T1], [H2, T2], ... [Hn, Tn]]))
	// can be used by light clients to make sure they've received the correct Txs
	txSha := types.DeriveSha(block.Transactions())
	if txSha != header.TxHash {
		err = fmt.Errorf("validating transaction root. received=%x got=%x", header.TxHash, txSha)
		return
	}

	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveSha(receipts)
	if receiptSha != header.ReceiptHash {
		err = fmt.Errorf("validating receipt root. received=%x got=%x", header.ReceiptHash, receiptSha)
		return
	}

	// Accumulate static rewards; block reward, uncle's and uncle inclusion.
	if err = sm.AccumulateRewards(state, block, parent); err != nil {
		return
	}

	// Commit state objects/accounts to a temporary trie (does not save)
	// used to calculate the state root.
	state.Update(common.Big0)
	if header.Root != state.Root() {
		err = fmt.Errorf("invalid merkle root. received=%x got=%x", header.Root, state.Root())
		return
	}

	// Calculate the td for this block
	td = CalculateTD(block, parent)
	// Sync the current block's state to the database
	state.Sync()
	// Remove transactions from the pool
	sm.txpool.RemoveSet(block.Transactions())

	for _, tx := range block.Transactions() {
		putTx(sm.extraDb, tx)
	}

	chainlogger.Infof("processed block #%d (%x...)\n", header.Number, block.Hash().Bytes()[0:4])

	return td, nil
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *BlockProcessor) ValidateHeader(block, parent *types.Header) error {
	if len(block.Extra) > 1024 {
		return fmt.Errorf("Block extra data too long (%d)", len(block.Extra))
	}

	expd := CalcDifficulty(block, parent)
	if expd.Cmp(block.Difficulty) != 0 {
		return fmt.Errorf("Difficulty check failed for block %v, %v", block.Difficulty, expd)
	}

	// block.gasLimit - parent.gasLimit <= parent.gasLimit / 1024
	a := new(big.Int).Sub(block.GasLimit, parent.GasLimit)
	b := new(big.Int).Div(parent.GasLimit, big.NewInt(1024))
	if a.Cmp(b) > 0 {
		return fmt.Errorf("GasLimit check failed for block %v (%v > %v)", block.GasLimit, a, b)
	}

	if block.Time <= parent.Time {
		return ValidationError("Block timestamp equal or less than previous block (%v - %v)", block.Time, parent.Time)
	}

	if int64(block.Time) > time.Now().Unix() {
		return BlockFutureErr
	}

	if new(big.Int).Sub(block.Number, parent.Number).Cmp(big.NewInt(1)) != 0 {
		return BlockNumberErr
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !sm.Pow.Verify(types.NewBlockWithHeader(block)) {
		return ValidationError("Block's nonce is invalid (= %x)", block.Nonce)
	}

	return nil
}

func (sm *BlockProcessor) AccumulateRewards(statedb *state.StateDB, block, parent *types.Block) error {
	reward := new(big.Int).Set(BlockReward)

	ancestors := set.New()
	uncles := set.New()
	ancestorHeaders := make(map[common.Hash]*types.Header)
	for _, ancestor := range sm.bc.GetAncestors(block, 7) {
		ancestorHeaders[ancestor.Hash()] = ancestor.Header()
		ancestors.Add(ancestor.Hash())
		// Include ancestors uncles in the uncle set. Uncles must be unique.
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
	}

	uncles.Add(block.Hash())
	for _, uncle := range block.Uncles() {
		if uncles.Has(uncle.Hash()) {
			// Error not unique
			return UncleError("Uncle not unique")
		}

		uncles.Add(uncle.Hash())

		if ancestors.Has(uncle.Hash()) {
			return UncleError("Uncle is ancestor")
		}

		if !ancestors.Has(uncle.ParentHash) {
			return UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
		}

		if err := sm.ValidateHeader(uncle, ancestorHeaders[uncle.ParentHash]); err != nil {
			return ValidationError(fmt.Sprintf("%v", err))
		}

		if !sm.Pow.Verify(types.NewBlockWithHeader(uncle)) {
			return ValidationError("Uncle's nonce is invalid (= %x)", uncle.Nonce)
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

func putTx(db common.Database, tx *types.Transaction) {
	rlpEnc, err := rlp.EncodeToBytes(tx)
	if err != nil {
		statelogger.Infoln("Failed encoding tx", err)
		return
	}
	db.Put(tx.Hash().Bytes(), rlpEnc)
}
