package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/pow/ezp"
	"github.com/ethereum/go-ethereum/state"
	"gopkg.in/fatih/set.v0"
)

var statelogger = logger.NewLogger("BLOCK")

type EthManager interface {
	BlockProcessor() *BlockProcessor
	ChainManager() *ChainManager
	TxPool() *TxPool
	PeerCount() int
	IsMining() bool
	IsListening() bool
	Peers() []*p2p.Peer
	KeyManager() *crypto.KeyManager
	ClientIdentity() p2p.ClientIdentity
	Db() ethutil.Database
	EventMux() *event.TypeMux
}

type BlockProcessor struct {
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

func NewBlockProcessor(txpool *TxPool, chainManager *ChainManager, eventMux *event.TypeMux) *BlockProcessor {
	sm := &BlockProcessor{
		mem:      make(map[string]*big.Int),
		Pow:      ezp.New(),
		bc:       chainManager,
		eventMux: eventMux,
		txpool:   txpool,
	}

	return sm
}

func (sm *BlockProcessor) TransitionState(statedb *state.StateDB, parent, block *types.Block) (receipts types.Receipts, err error) {
	coinbase := statedb.GetOrNewStateObject(block.Header().Coinbase)
	coinbase.SetGasPool(CalcGasLimit(parent, block))

	// Process the transactions on to current block
	receipts, _, _, _, err = sm.ApplyTransactions(coinbase, statedb, block, block.Transactions(), false)
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (self *BlockProcessor) ApplyTransactions(coinbase *state.StateObject, state *state.StateDB, block *types.Block, txs types.Transactions, transientProcess bool) (types.Receipts, types.Transactions, types.Transactions, types.Transactions, error) {
	var (
		receipts           types.Receipts
		handled, unhandled types.Transactions
		erroneous          types.Transactions
		totalUsedGas       = big.NewInt(0)
		err                error
		cumulativeSum      = new(big.Int)
	)

done:
	for i, tx := range txs {
		// If we are mining this block and validating we want to set the logs back to 0
		state.EmptyLogs()

		txGas := new(big.Int).Set(tx.Gas())

		cb := state.GetStateObject(coinbase.Address())
		st := NewStateTransition(NewEnv(state, self.bc, tx, block), tx, cb)
		_, err = st.TransitionState()
		if err != nil {
			switch {
			case IsNonceErr(err):
				err = nil // ignore error
				continue
			case IsGasLimitErr(err):
				unhandled = txs[i:]

				break done
			default:
				statelogger.Infoln(err)
				erroneous = append(erroneous, tx)
				err = nil
			}
		}

		txGas.Sub(txGas, st.gas)
		cumulativeSum.Add(cumulativeSum, new(big.Int).Mul(txGas, tx.GasPrice()))

		// Update the state with pending changes
		state.Update(txGas)

		cumulative := new(big.Int).Set(totalUsedGas.Add(totalUsedGas, txGas))
		receipt := types.NewReceipt(state.Root(), cumulative)
		receipt.SetLogs(state.Logs())
		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
		chainlogger.Debugln(receipt)

		// Notify all subscribers
		if !transientProcess {
			go self.eventMux.Post(TxPostEvent{tx})
		}

		receipts = append(receipts, receipt)
		handled = append(handled, tx)

		if ethutil.Config.Diff && ethutil.Config.DiffType == "all" {
			state.CreateOutputForDiff()
		}
	}

	block.Reward = cumulativeSum
	block.Header().GasUsed = totalUsedGas

	return receipts, handled, unhandled, erroneous, err
}

func (sm *BlockProcessor) Process(block *types.Block) (td *big.Int, msgs state.Messages, err error) {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	header := block.Header()
	if sm.bc.HasBlock(header.Hash()) {
		return nil, nil, &KnownBlockError{header.Number, header.Hash()}
	}

	if !sm.bc.HasBlock(header.ParentHash) {
		return nil, nil, ParentError(header.ParentHash)
	}
	parent := sm.bc.GetBlock(header.ParentHash)

	return sm.ProcessWithParent(block, parent)
}

func (sm *BlockProcessor) ProcessWithParent(block, parent *types.Block) (td *big.Int, messages state.Messages, err error) {
	sm.lastAttemptedBlock = block

	state := state.New(parent.Trie().Copy())

	// Block validation
	if err = sm.ValidateBlock(block, parent); err != nil {
		return
	}

	receipts, err := sm.TransitionState(state, parent, block)
	if err != nil {
		return
	}

	header := block.Header()

	rbloom := types.CreateBloom(receipts)
	if bytes.Compare(rbloom, header.Bloom) != 0 {
		err = fmt.Errorf("unable to replicate block's bloom=%x", rbloom)
		return
	}

	txSha := types.DeriveSha(block.Transactions())
	if bytes.Compare(txSha, header.TxHash) != 0 {
		err = fmt.Errorf("validating transaction root. received=%x got=%x", header.TxHash, txSha)
		return
	}

	receiptSha := types.DeriveSha(receipts)
	if bytes.Compare(receiptSha, header.ReceiptHash) != 0 {
		fmt.Println("receipts", receipts)
		err = fmt.Errorf("validating receipt root. received=%x got=%x", header.ReceiptHash, receiptSha)
		return
	}

	if err = sm.AccumelateRewards(state, block, parent); err != nil {
		return
	}

	state.Update(ethutil.Big0)

	if !bytes.Equal(header.Root, state.Root()) {
		err = fmt.Errorf("invalid merkle root. received=%x got=%x", header.Root, state.Root())
		return
	}

	// Calculate the new total difficulty and sync back to the db
	if td, ok := sm.CalculateTD(block); ok {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		state.Sync()

		state.Manifest().SetHash(block.Hash())

		messages := state.Manifest().Messages
		state.Manifest().Reset()

		chainlogger.Infof("Processed block #%d (%x...)\n", header.Number, block.Hash()[0:4])

		sm.txpool.RemoveSet(block.Transactions())

		return td, messages, nil
	} else {
		return nil, nil, errors.New("total diff failed")
	}
}

func (sm *BlockProcessor) CalculateTD(block *types.Block) (*big.Int, bool) {
	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles() {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	// TD(genesis_block) = 0 and TD(B) = TD(B.parent) + sum(u.difficulty for u in B.uncles) + B.difficulty
	td := new(big.Int)
	td = td.Add(sm.bc.Td(), uncleDiff)
	td = td.Add(td, block.Header().Difficulty)

	// The new TD will only be accepted if the new difficulty is
	// is greater than the previous.
	if td.Cmp(sm.bc.Td()) > 0 {
		return td, true
	}

	return nil, false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *BlockProcessor) ValidateBlock(block, parent *types.Block) error {
	expd := CalcDifficulty(block, parent)
	if expd.Cmp(block.Header().Difficulty) < 0 {
		return fmt.Errorf("Difficulty check failed for block %v, %v", block.Header().Difficulty, expd)
	}

	diff := block.Header().Time - parent.Header().Time
	if diff < 0 {
		return ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Header().Time, sm.bc.CurrentBlock().Header().Time)
	}

	/* XXX
	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}
	*/

	// Verify the nonce of the block. Return an error if it's not valid
	if !sm.Pow.Verify(block /*block.HashNoNonce(), block.Difficulty, block.Nonce*/) {
		return ValidationError("Block's nonce is invalid (= %v)", ethutil.Bytes2Hex(block.Header().Nonce))
	}

	return nil
}

func (sm *BlockProcessor) AccumelateRewards(statedb *state.StateDB, block, parent *types.Block) error {
	reward := new(big.Int).Set(BlockReward)

	knownUncles := set.New()
	for _, uncle := range parent.Uncles() {
		knownUncles.Add(string(uncle.Hash()))
	}

	nonces := ethutil.NewSet(block.Header().Nonce)
	for _, uncle := range block.Uncles() {
		if nonces.Include(uncle.Nonce) {
			// Error not unique
			return UncleError("Uncle not unique")
		}

		uncleParent := sm.bc.GetBlock(uncle.ParentHash)
		if uncleParent == nil {
			return UncleError(fmt.Sprintf("Uncle's parent unknown (%x)", uncle.ParentHash[0:4]))
		}

		if uncleParent.Header().Number.Cmp(new(big.Int).Sub(parent.Header().Number, big.NewInt(6))) < 0 {
			return UncleError("Uncle too old")
		}

		if knownUncles.Has(string(uncle.Hash())) {
			return UncleError("Uncle in chain")
		}

		nonces.Insert(uncle.Nonce)

		r := new(big.Int)
		r.Mul(BlockReward, big.NewInt(15)).Div(r, big.NewInt(16))

		uncleAccount := statedb.GetAccount(uncle.Coinbase)
		uncleAccount.AddAmount(r)

		reward.Add(reward, new(big.Int).Div(BlockReward, big.NewInt(32)))
	}

	// Get the account associated with the coinbase
	account := statedb.GetAccount(block.Header().Coinbase)
	// Reward amount of ether to the coinbase address
	account.AddAmount(reward)

	statedb.Manifest().AddMessage(&state.Message{
		To:        block.Header().Coinbase,
		Input:     nil,
		Origin:    nil,
		Timestamp: int64(block.Header().Time), Coinbase: block.Header().Coinbase, Number: block.Header().Number,
		Value: new(big.Int).Add(reward, block.Reward),
	})

	return nil
}

func (sm *BlockProcessor) GetMessages(block *types.Block) (messages []*state.Message, err error) {
	if !sm.bc.HasBlock(block.Header().ParentHash) {
		return nil, ParentError(block.Header().ParentHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.Header().ParentHash)
		state  = state.New(parent.Trie().Copy())
	)

	defer state.Reset()

	sm.TransitionState(state, parent, block)
	sm.AccumelateRewards(state, block, parent)

	return state.Manifest().Messages, nil
}
