package core

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
)

const (
	// must be bumped when consensus algorithm is changed, this forces the upgradedb
	// command to be run (forces the blocks to be imported again using the new algorithm)
	BlockChainVersion = 3
)

var receiptsPre = []byte("receipts-")

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

	events event.Subscription

	eventMux *event.TypeMux
}

func NewBlockProcessor(db, extra common.Database, pow pow.PoW, chainManager *ChainManager, eventMux *event.TypeMux) *BlockProcessor {
	sm := &BlockProcessor{
		db:       db,
		extraDb:  extra,
		mem:      make(map[string]*big.Int),
		Pow:      pow,
		bc:       chainManager,
		eventMux: eventMux,
	}

	return sm
}

func (sm *BlockProcessor) TransitionState(statedb *state.StateDB, parent, block *types.Block, transientProcess bool) (receipts types.Receipts, err error) {
	coinbase := statedb.GetOrNewStateObject(block.Header().Coinbase)
	coinbase.SetGasLimit(block.Header().GasLimit)

	// Process the transactions on to parent state
	receipts, err = sm.ApplyTransactions(coinbase, statedb, block, block.Transactions(), transientProcess)
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

func (self *BlockProcessor) ApplyTransaction(coinbase *state.StateObject, statedb *state.StateDB, block *types.Block, tx *types.Transaction, usedGas *big.Int, transientProcess bool) (*types.Receipt, *big.Int, error) {
	// If we are mining this block and validating we want to set the logs back to 0

	cb := statedb.GetStateObject(coinbase.Address())
	_, gas, err := ApplyMessage(NewEnv(statedb, self.bc, tx, block), tx, cb)
	if err != nil && (IsNonceErr(err) || state.IsGasLimitErr(err) || IsInvalidTxErr(err)) {
		return nil, nil, err
	}

	// Update the state with pending changes
	statedb.Update()

	cumulative := new(big.Int).Set(usedGas.Add(usedGas, gas))
	receipt := types.NewReceipt(statedb.Root().Bytes(), cumulative)

	logs := statedb.GetLogs(tx.Hash())
	receipt.SetLogs(logs)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	glog.V(logger.Debug).Infoln(receipt)

	// Notify all subscribers
	if !transientProcess {
		go self.eventMux.Post(TxPostEvent{tx})
		go self.eventMux.Post(logs)
	}

	return receipt, gas, err
}
func (self *BlockProcessor) ChainManager() *ChainManager {
	return self.bc
}

func (self *BlockProcessor) ApplyTransactions(coinbase *state.StateObject, statedb *state.StateDB, block *types.Block, txs types.Transactions, transientProcess bool) (types.Receipts, error) {
	var (
		receipts      types.Receipts
		totalUsedGas  = big.NewInt(0)
		err           error
		cumulativeSum = new(big.Int)
	)

	for i, tx := range txs {
		statedb.StartRecord(tx.Hash(), block.Hash(), i)

		receipt, txGas, err := self.ApplyTransaction(coinbase, statedb, block, tx, totalUsedGas, transientProcess)
		if err != nil && (IsNonceErr(err) || state.IsGasLimitErr(err) || IsInvalidTxErr(err)) {
			return nil, err
		}

		if err != nil {
			glog.V(logger.Core).Infoln("TX err:", err)
		}
		receipts = append(receipts, receipt)

		cumulativeSum.Add(cumulativeSum, new(big.Int).Mul(txGas, tx.GasPrice()))
	}

	if block.GasUsed().Cmp(totalUsedGas) != 0 {
		return nil, ValidationError(fmt.Sprintf("gas used error (%v / %v)", block.GasUsed(), totalUsedGas))
	}

	if transientProcess {
		go self.eventMux.Post(PendingBlockEvent{block, statedb.Logs()})
	}

	return receipts, err
}

func (sm *BlockProcessor) RetryProcess(block *types.Block) (logs state.Logs, err error) {
	// Processing a blocks may never happen simultaneously
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	header := block.Header()
	if !sm.bc.HasBlock(header.ParentHash) {
		return nil, ParentError(header.ParentHash)
	}
	parent := sm.bc.GetBlock(header.ParentHash)

	// FIXME Change to full header validation. See #1225
	errch := make(chan bool)
	go func() { errch <- sm.Pow.Verify(block) }()

	logs, err = sm.processWithParent(block, parent)
	if !<-errch {
		return nil, ValidationError("Block's nonce is invalid (= %x)", block.Nonce)
	}

	return logs, err
}

// Process block will attempt to process the given block's transactions and applies them
// on top of the block's parent state (given it exists) and will return wether it was
// successful or not.
func (sm *BlockProcessor) Process(block *types.Block) (logs state.Logs, err error) {
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

func (sm *BlockProcessor) processWithParent(block, parent *types.Block) (logs state.Logs, err error) {
	// Create a new state based on the parent's root (e.g., create copy)
	state := state.New(parent.Root(), sm.db)

	// Block validation
	if err = ValidateHeader(sm.Pow, block.Header(), parent.Header(), false); err != nil {
		return
	}

	// There can be at most two uncles
	if len(block.Uncles()) > 2 {
		return nil, ValidationError("Block can only contain maximum 2 uncles (contained %v)", len(block.Uncles()))
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

	// The transactions Trie's root (R = (Tr [[i, RLP(T1)], [i, RLP(T2)], ... [n, RLP(Tn)]]))
	// can be used by light clients to make sure they've received the correct Txs
	txSha := types.DeriveSha(block.Transactions())
	if txSha != header.TxHash {
		err = fmt.Errorf("invalid transaction root hash. received=%x calculated=%x", header.TxHash, txSha)
		return
	}

	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveSha(receipts)
	if receiptSha != header.ReceiptHash {
		err = fmt.Errorf("invalid receipt root hash. received=%x calculated=%x", header.ReceiptHash, receiptSha)
		return
	}

	// Verify UncleHash before running other uncle validations
	unclesSha := block.CalculateUnclesHash()
	if unclesSha != header.UncleHash {
		err = fmt.Errorf("invalid uncles root hash. received=%x calculated=%x", header.UncleHash, unclesSha)
		return
	}

	// Verify uncles
	if err = sm.VerifyUncles(state, block, parent); err != nil {
		return
	}
	// Accumulate static rewards; block reward, uncle's and uncle inclusion.
	AccumulateRewards(state, block)

	// Commit state objects/accounts to a temporary trie (does not save)
	// used to calculate the state root.
	state.Update()
	if header.Root != state.Root() {
		err = fmt.Errorf("invalid merkle root. received=%x got=%x", header.Root, state.Root())
		return
	}

	// Sync the current block's state to the database
	state.Sync()

	// This puts transactions in a extra db for rpc
	for i, tx := range block.Transactions() {
		putTx(sm.extraDb, tx, block, uint64(i))
	}

	// store the receipts
	putReceipts(sm.extraDb, block.Hash(), receipts)

	return state.Logs(), nil
}

func (sm *BlockProcessor) VerifyUncles(statedb *state.StateDB, block, parent *types.Block) error {
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
	for i, uncle := range block.Uncles() {
		hash := uncle.Hash()
		if uncles.Has(hash) {
			// Error not unique
			return UncleError("uncle[%d](%x) not unique", i, hash[:4])
		}
		uncles.Add(hash)

		if ancestors.Has(hash) {
			branch := fmt.Sprintf("  O - %x\n  |\n", block.Hash())
			ancestors.Each(func(item interface{}) bool {
				branch += fmt.Sprintf("  O - %x\n  |\n", hash)
				return true
			})
			glog.Infoln(branch)

			return UncleError("uncle[%d](%x) is ancestor", i, hash[:4])
		}

		if !ancestors.Has(uncle.ParentHash) || uncle.ParentHash == parent.Hash() {
			return UncleError("uncle[%d](%x)'s parent is not ancestor (%x)", i, hash[:4], uncle.ParentHash[0:4])
		}

		if err := ValidateHeader(sm.Pow, uncle, ancestorHeaders[uncle.ParentHash], true); err != nil {
			return ValidationError(fmt.Sprintf("uncle[%d](%x) header invalid: %v", i, hash[:4], err))
		}
	}

	return nil
}

// GetBlockReceipts returns the receipts beloniging to the block hash
func (sm *BlockProcessor) GetBlockReceipts(bhash common.Hash) (receipts types.Receipts, err error) {
	return getBlockReceipts(sm.extraDb, bhash)
}

// GetLogs returns the logs of the given block. This method is using a two step approach
// where it tries to get it from the (updated) method which gets them from the receipts or
// the depricated way by re-processing the block.
func (sm *BlockProcessor) GetLogs(block *types.Block) (logs state.Logs, err error) {
	receipts, err := sm.GetBlockReceipts(block.Hash())
	if err == nil && len(receipts) > 0 {
		// coalesce logs
		for _, receipt := range receipts {
			logs = append(logs, receipt.Logs()...)
		}
		return
	}

	// TODO: remove backward compatibility
	var (
		parent = sm.bc.GetBlock(block.Header().ParentHash)
		state  = state.New(parent.Root(), sm.db)
	)

	sm.TransitionState(state, parent, block, true)

	return state.Logs(), nil
}

// See YP section 4.3.4. "Block Header Validity"
// Validates a block. Returns an error if the block is invalid.
func ValidateHeader(pow pow.PoW, block, parent *types.Header, checkPow bool) error {
	if big.NewInt(int64(len(block.Extra))).Cmp(params.MaximumExtraDataSize) == 1 {
		return fmt.Errorf("Block extra data too long (%d)", len(block.Extra))
	}

	expd := CalcDifficulty(block, parent)
	if expd.Cmp(block.Difficulty) != 0 {
		return fmt.Errorf("Difficulty check failed for block %v, %v", block.Difficulty, expd)
	}

	a := new(big.Int).Sub(block.GasLimit, parent.GasLimit)
	a.Abs(a)
	b := new(big.Int).Div(parent.GasLimit, params.GasLimitBoundDivisor)
	if !(a.Cmp(b) < 0) || (block.GasLimit.Cmp(params.MinGasLimit) == -1) {
		return fmt.Errorf("GasLimit check failed for block %v (%v > %v)", block.GasLimit, a, b)
	}

	if int64(block.Time) > time.Now().Unix() {
		return BlockFutureErr
	}

	if new(big.Int).Sub(block.Number, parent.Number).Cmp(big.NewInt(1)) != 0 {
		return BlockNumberErr
	}

	if block.Time <= parent.Time {
		return BlockEqualTSErr //ValidationError("Block timestamp equal or less than previous block (%v - %v)", block.Time, parent.Time)
	}

	if checkPow {
		// Verify the nonce of the block. Return an error if it's not valid
		if !pow.Verify(types.NewBlockWithHeader(block)) {
			return ValidationError("Block's nonce is invalid (= %x)", block.Nonce)
		}
	}

	return nil
}

func AccumulateRewards(statedb *state.StateDB, block *types.Block) {
	reward := new(big.Int).Set(BlockReward)

	for _, uncle := range block.Uncles() {
		num := new(big.Int).Add(big.NewInt(8), uncle.Number)
		num.Sub(num, block.Number())

		r := new(big.Int)
		r.Mul(BlockReward, num)
		r.Div(r, big.NewInt(8))

		statedb.AddBalance(uncle.Coinbase, r)

		reward.Add(reward, new(big.Int).Div(BlockReward, big.NewInt(32)))
	}

	// Get the account associated with the coinbase
	statedb.AddBalance(block.Header().Coinbase, reward)
}

func getBlockReceipts(db common.Database, bhash common.Hash) (receipts types.Receipts, err error) {
	var rdata []byte
	rdata, err = db.Get(append(receiptsPre, bhash[:]...))

	if err == nil {
		err = rlp.DecodeBytes(rdata, &receipts)
	} else {
		glog.V(logger.Detail).Infof("getBlockReceipts error %v\n", err)
	}
	return
}

func putTx(db common.Database, tx *types.Transaction, block *types.Block, i uint64) {
	rlpEnc, err := rlp.EncodeToBytes(tx)
	if err != nil {
		glog.V(logger.Debug).Infoln("Failed encoding tx", err)
		return
	}
	db.Put(tx.Hash().Bytes(), rlpEnc)

	var txExtra struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}
	txExtra.BlockHash = block.Hash()
	txExtra.BlockIndex = block.NumberU64()
	txExtra.Index = i
	rlpMeta, err := rlp.EncodeToBytes(txExtra)
	if err != nil {
		glog.V(logger.Debug).Infoln("Failed encoding tx meta data", err)
		return
	}
	db.Put(append(tx.Hash().Bytes(), 0x0001), rlpMeta)
}

func putReceipts(db common.Database, hash common.Hash, receipts types.Receipts) error {
	storageReceipts := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		storageReceipts[i] = (*types.ReceiptForStorage)(receipt)
	}

	bytes, err := rlp.EncodeToBytes(storageReceipts)
	if err != nil {
		return err
	}

	db.Put(append(receiptsPre, hash[:]...), bytes)

	return nil
}
