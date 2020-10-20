// Copyright 2020 The go-ethereum Authors
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

package lotterybook

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/merkletree"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

const defaultCostLifeTime = time.Hour * 24

type cost struct {
	timestamp time.Time
	value     *big.Float
}

type costWatcher struct {
	lock     sync.Mutex
	lifeTime time.Duration
	costs    []*cost
	avg      *big.Float // Cached average cost, will be invalidate if cost list changes
}

func newCostWatcher(lifeTime time.Duration) *costWatcher {
	return &costWatcher{lifeTime: lifeTime}
}

func (cw *costWatcher) prune() {
	// Assume the lock is held.
	var pruned = -1
	for index, c := range cw.costs {
		if time.Since(c.timestamp) > cw.lifeTime {
			pruned = index
			continue
		}
		break
	}
	if pruned != -1 {
		cw.costs = cw.costs[pruned+1:]
		cw.avg = nil
	}
}

func (cw *costWatcher) add(price *big.Int, limit *big.Int) {
	feesWei := new(big.Int).Mul(limit, price)
	feesEth := new(big.Float).Quo(new(big.Float).SetInt(feesWei), new(big.Float).SetInt(big.NewInt(params.Ether)))

	cw.lock.Lock()
	cw.costs = append(cw.costs, &cost{
		timestamp: time.Now(),
		value:     feesEth,
	})
	cw.avg = nil
	cw.prune()
	cw.lock.Unlock()
}

func (cw *costWatcher) average() *big.Float {
	cw.lock.Lock()
	defer cw.lock.Unlock()

	cw.prune()
	if cw.avg != nil {
		return cw.avg
	}
	if len(cw.costs) == 0 {
		return nil
	}
	total := new(big.Float)
	for _, c := range cw.costs {
		total.Add(total, c.value)
	}
	cw.avg = new(big.Float).Quo(total, big.NewFloat(float64(len(cw.costs))))
	return cw.avg
}

// ChequeDrawer represents the payment drawer in a off-chain payment channel.
// Usually in LES protocol the drawer refers to a light client.
//
// ChequeDrawer is self-contained and stateful, it will only offer the most
// basic function: Issue, Deposit and some relevant query APIs.
//
// Internally it relies on lottery manager for lottery life cycle management.
//
// There is an assumption held that the local chain is synced. Otherwise these
// scenarios can happen:
// - Drawer needs to wait very long time before an on-chain transaction be
//   confirmed(create/reset/destory lottery) which eventually lead to timeout
//   error.
// - Send invalid cheques based on the revealed(even claimed/resetted/destoryed
//   lottery). It may lead to a network termination.
type ChequeDrawer struct {
	address  common.Address
	cdb      *chequeDB
	book     *LotteryBook
	chain    Blockchain
	lmgr     *lotteryManager
	cBackend bind.ContractBackend
	dBackend bind.DeployBackend
	rand     *rand.Rand
	cw       *costWatcher

	txSigner     *bind.TransactOpts                // Used for production environment, transaction signer
	keySigner    func(data []byte) ([]byte, error) // Used for testing, cheque signer
	chequeSigner func(data []byte) ([]byte, error) // Used for production environment, cheque signer
}

// NewChequeDrawer creates a payment drawer and deploys the contract if necessary.
func NewChequeDrawer(address, contractAddr common.Address, txSigner *bind.TransactOpts, chequeSigner func(data []byte) ([]byte, error), chain Blockchain, cBackend bind.ContractBackend, dBackend bind.DeployBackend, db ethdb.Database) (*ChequeDrawer, error) {
	if contractAddr == (common.Address{}) {
		return nil, errors.New("empty contract address")
	}
	book, err := newLotteryBook(contractAddr, cBackend)
	if err != nil {
		return nil, err
	}
	cdb := newChequeDB(db)
	drawer := &ChequeDrawer{
		address:      address,
		cdb:          cdb,
		book:         book,
		txSigner:     txSigner,
		chequeSigner: chequeSigner,
		chain:        chain,
		cBackend:     cBackend,
		dBackend:     dBackend,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
		cw:           newCostWatcher(defaultCostLifeTime),
	}
	drawer.lmgr = newLotteryManager(address, chain, book.contract, cdb, drawer.wipeLottery, drawer.writeLottery)
	return drawer, nil
}

// ContractAddr returns the address of deployed accountbook contract.
func (drawer *ChequeDrawer) ContractAddr() common.Address {
	return drawer.book.address
}

// Close exits all background threads and closes all event subscribers.
func (drawer *ChequeDrawer) Close() {
	drawer.lmgr.close()
}

// newProbabilityTree constructs a probability tree(merkle tree) based on the
// given list of receivers and corresponding amounts. The payment amount can
// used as the initial weight for each payee. Since the underlying merkle tree
// is binary tree, so finally all weights will be adjusted to 1/2^N form.
//
// If some entries are assigned with a very small weight, then it may not included
// in the tree. In this case, kick the relevant payee out.
func (drawer *ChequeDrawer) newProbabilityTree(payees []common.Address, amounts []uint64) (*merkletree.MerkleTree, []*merkletree.Entry, map[string]struct{}, uint64) {
	var totalAmount uint64
	for _, amount := range amounts {
		totalAmount += amount
	}
	entries := make([]*merkletree.Entry, len(payees))
	for index, amount := range amounts {
		entries[index] = &merkletree.Entry{
			Value:  payees[index].Bytes(),
			Weight: amount,
		}
	}
	tree, dropped := merkletree.NewMerkleTree(entries)
	return tree, entries, dropped, totalAmount
}

// submitLottery creates the lottery based on the specified batch of payees and
// corresponding payment amount. Return the newly created cheque list.
func (drawer *ChequeDrawer) submitLottery(context context.Context, payees []common.Address, amounts []uint64, revealNumber uint64, onchainFn func(context context.Context, amount uint64, id [32]byte, blockNumber uint64, salt uint64) (*types.Transaction, error)) (*Lottery, error) {
	if len(payees) != len(amounts) {
		return nil, errors.New("inconsistent payment receivers and amounts")
	}
	if len(amounts) == 0 {
		return nil, errors.New("empty payment list")
	}
	// Construct merkle probability tree with given payer list and corresponding weight
	tree, entries, dropped, amount := drawer.newProbabilityTree(payees, amounts)

	// New random lottery salt to ensure the id is unique.
	salt := drawer.rand.Uint64()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, salt)

	if len(dropped) > 0 {
		var removed []int
		for index, payee := range payees {
			if _, ok := dropped[string(payee.Bytes())]; ok {
				removed = append(removed, index)
			}
		}
		for i := 0; i < len(removed); i++ {
			payees = append(payees[:removed[i]-i], payees[removed[i]-i+1:]...)
		}
	}
	// Submit the new created lottery to contract by specified on-chain function.
	lotteryId := crypto.Keccak256Hash(append(tree.Hash().Bytes(), buf...))
	start := time.Now()
	tx, err := onchainFn(context, amount, lotteryId, revealNumber, salt)
	if err != nil {
		return nil, err
	}
	// Generate and store the temporary lottery record before the transaction
	// confirmation. If any crash happen then we can still claim back the deposit
	// inside. But if after we sending out the tx and then crash happens, in
	// this case we can't do anything here.
	lottery := &Lottery{
		Id:           lotteryId,
		RevealNumber: revealNumber,
		Amount:       amount,
		Receivers:    payees,
		GasPrice:     tx.GasPrice(),
		Nonce:        tx.Nonce(),
		// Leave the CreateAt as empty.
	}
	// Generate empty unused cheques based on the newly created lottery.
	for _, entry := range entries {
		if _, ok := dropped[string(entry.Value)]; ok {
			continue
		}
		witness, err := tree.Prove(entry)
		if err != nil {
			return nil, err
		}
		cheque, err := newCheque(witness, drawer.ContractAddr(), salt, entry.Salt())
		if err != nil {
			return nil, err
		}
		if drawer.keySigner != nil {
			// If it's testing, use provided key signer.
			if err := cheque.signWithKey(drawer.keySigner); err != nil {
				return nil, err
			}
		} else {
			// Otherwise, use provided clef as the production-environment signer.
			if err := cheque.sign(drawer.chequeSigner); err != nil {
				return nil, err
			}
		}
		drawer.cdb.writeCheque(common.BytesToAddress(entry.Value), drawer.address, cheque, true)
	}
	drawer.cdb.writeLottery(drawer.address, lotteryId, true, lottery) // tmp = true

	// WARN: this procedure may take very long time.
	receipt, err := bind.WaitMined(context, drawer.dBackend, tx)
	if err != nil {
		return nil, NewErrTransactionNotConfirmed(lotteryId, tx.Nonce(), tx.GasPrice())
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, ErrTransactionFailed
	}
	depositDurationTimer.UpdateSince(start)

	lottery.CreateAt = receipt.BlockNumber.Uint64()                    // Assign the created block number
	drawer.cdb.writeLottery(drawer.address, lotteryId, false, lottery) // Store the lottery after cheques
	drawer.cdb.deleteLottery(drawer.address, lotteryId, true)          // Now we can sunset the tmp record

	drawer.cw.add(tx.GasPrice(), big.NewInt(int64(receipt.GasUsed)))
	return lottery, nil
}

// createLottery creates the lottery based on the specified batch of payees and
// corresponding payment amount, returns the id of craeted lottery.
func (drawer *ChequeDrawer) createLottery(ctx context.Context, payees []common.Address, amounts []uint64, revealNumber uint64, nonce *uint64, gasprice *big.Int) (common.Hash, error) {
	onchainFn := func(ctx context.Context, amount uint64, id [32]byte, blockNumber uint64, salt uint64) (*types.Transaction, error) {
		// Create an independent auth opt to submit the lottery
		opt := &bind.TransactOpts{
			Context: ctx,
			From:    drawer.address,
			Signer:  drawer.txSigner.Signer,
			Value:   big.NewInt(int64(amount)),
		}
		if nonce != nil {
			opt.Nonce = big.NewInt(int64(*nonce))
		}
		if gasprice != nil {
			opt.GasPrice = gasprice
		}
		return drawer.book.contract.NewLottery(opt, id, blockNumber, salt)
	}
	lottery, err := drawer.submitLottery(ctx, payees, amounts, revealNumber, onchainFn)
	if err != nil {
		return common.Hash{}, err
	}
	if err := drawer.lmgr.trackLottery(lottery); err != nil {
		return common.Hash{}, err
	}
	createLotteryGauge.Inc(1)
	return lottery.Id, err
}

// resetLottery resets a existed stale lottery with new batch of payment receivers
// and corresponding amount. Add more funds into lottery ff the deposit of stale
// lottery is not enough to cover the new amount. Otherwise the deposit given by
// lottery for each receiver may be higher than the specified value.
func (drawer *ChequeDrawer) resetLottery(ctx context.Context, id common.Hash, payees []common.Address, amounts []uint64, revealNumber uint64, nonce *uint64, gasprice *big.Int) (common.Hash, error) {
	// Short circuit if the specified stale lottery doesn't exist.
	lottery := drawer.cdb.readLottery(drawer.address, id)
	if lottery == nil {
		return common.Hash{}, errors.New("the lottery specified is not-existent")
	}
	onchainFn := func(ctx context.Context, amount uint64, newid [32]byte, blockNumber uint64, salt uint64) (*types.Transaction, error) {
		// Create an independent auth opt to submit the lottery
		var netAmount uint64
		if lottery.Amount < amount {
			netAmount = amount - lottery.Amount
		}
		opt := &bind.TransactOpts{
			Context: ctx,
			From:    drawer.address,
			Signer:  drawer.txSigner.Signer,
			Value:   big.NewInt(int64(netAmount)),
		}
		if nonce != nil {
			opt.Nonce = big.NewInt(int64(*nonce))
		}
		if gasprice != nil {
			opt.GasPrice = gasprice
		}
		// The on-chain transaction may fail dues to lots of reasons like
		// the lottery doesn't exist or lottery hasn't been expired yet.
		return drawer.book.contract.ResetLottery(opt, id, newid, blockNumber, salt)
	}
	newLottery, err := drawer.submitLottery(ctx, payees, amounts, revealNumber, onchainFn)
	if err != nil {
		return common.Hash{}, err
	}
	// Update manager's status
	if err := drawer.lmgr.trackLottery(newLottery); err != nil {
		return common.Hash{}, err
	}
	if err := drawer.lmgr.deleteExpired(id); err != nil {
		return common.Hash{}, err
	}
	reownLotteryGauge.Inc(1)
	return newLottery.Id, nil
}

// Deposit is a wrapper function of `createLottery` and `resetLottery`.
// The strategy of deposit here is very simple: if there are any expired lotteries
// can be reowned, use these lottery first; otherwise create new lottery for deposit.
func (drawer *ChequeDrawer) Deposit(context context.Context, payees []common.Address, amounts []uint64, revealNumber uint64, nonce *uint64, gasprice *big.Int) (common.Hash, error) {
	expired, err := drawer.lmgr.expiredLotteries()
	if err != nil {
		return common.Hash{}, err
	}
	// We have some expired lottery can be reused, don't create a fresh new here.
	if len(expired) > 0 {
		// Select a lottery whose amount is closest to the target amount to reset.
		var (
			total  uint64
			bias   uint64
			picked common.Hash
		)
		for _, amount := range amounts {
			total += amount
		}
		for _, l := range expired {
			var b uint64
			if l.Amount < total {
				b = total - l.Amount
			} else {
				b = l.Amount - total
			}
			if bias == 0 || bias > b {
				bias, picked = b, l.Id
			}
		}
		return drawer.resetLottery(context, picked, payees, amounts, revealNumber, nonce, gasprice)
	}
	// Nothing can be reused, create a new lottery for payment
	return drawer.createLottery(context, payees, amounts, revealNumber, nonce, gasprice)
}

// destroyLottery destroys a stale lottery, claims all deposit inside back to
// our own pocket.
func (drawer *ChequeDrawer) destroyLottery(context context.Context, id common.Hash) error {
	// Short circuit if the specified stale lottery doesn't exist.
	lottery := drawer.cdb.readLottery(drawer.address, id)
	if lottery == nil {
		return errors.New("the lottery specified is not-existent")
	}
	// The on-chain transaction may fail dues to lots of reasons like
	// the lottery doesn't exist or lottery hasn't been expired yet.
	start := time.Now()
	tx, err := drawer.book.contract.DestroyLottery(drawer.txSigner, id)
	if err != nil {
		return err
	}
	receipt, err := bind.WaitMined(context, drawer.dBackend, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return ErrTransactionFailed
	}
	destroyDurationTimer.UpdateSince(start)
	if err := drawer.lmgr.deleteExpired(id); err != nil {
		return err
	}
	return nil
}

// Destroy is a wrapper function of destroyLottery. It destorys all
// expired lotteries and claim back all deposit inside.
func (drawer *ChequeDrawer) Destroy(context context.Context) error {
	expired, err := drawer.lmgr.expiredLotteries()
	if err != nil {
		return err
	}
	reownLotteryGauge.Inc(1)
	for _, l := range expired {
		drawer.destroyLottery(context, l.Id)
	}
	return nil
}

// IssueCheque creates a cheque for issuing specified amount for payee.
//
// Many active lotteries can be used to create cheque, we use the simplest
// strategy here for lottery selection: choose a lottery ticket with the
// most recent expiration date and the remaining amount can cover the amount
// paid this time.
func (drawer *ChequeDrawer) IssueCheque(payee common.Address, amount uint64) ([]*Cheque, error) {
	if amount == 0 {
		return nil, errors.New("invalid amount")
	}
	lotteries, err := drawer.lmgr.activeLotteries()
	if err != nil {
		return nil, err
	}
	sort.Sort(LotteryByRevealTime(lotteries))

	var (
		cheques  []*Cheque
		remained = amount
	)
	for _, lottery := range lotteries {
		// Short circuit if the lottery doesn't contain the target payee.
		if !lottery.hasReceiver(payee) {
			continue
		}
		// We have another additional check here(but it's optional).
		// If the reveal time is VERY close, then don't use it anymore.
		// Seems (1) our chain may lag behind (2) when the receiver
		// gets this cheque, the lottery might expire at that time.
		// So leave us a few safe time range.
		current := drawer.chain.CurrentHeader().Number.Uint64()
		if current+2*lotterySafetyMargin >= lottery.RevealNumber {
			continue
		}
		cheque := drawer.cdb.readCheque(payee, drawer.address, lottery.Id, true)
		if cheque == nil {
			continue
		}
		// If the remaining allowance is enough to cover the expense, stop
		// iteration. Otherwise find more.
		allowance := lottery.balance(payee, cheque)
		if allowance >= remained {
			cheque, err := drawer.issueCheque(payee, lottery.Id, remained, false)
			if err != nil {
				continue
			}
			cheques = append(cheques, cheque)
			remained = 0
		} else if allowance != 0 {
			cheque, err := drawer.issueCheque(payee, lottery.Id, allowance, false)
			if err != nil {
				continue
			}
			cheques = append(cheques, cheque)
			remained -= allowance
		}
		if remained == 0 {
			break
		}
	}
	if remained != 0 {
		return nil, ErrNotEnoughDeposit // No suitable lotteries found for payment
	}
	// Finally persist all issued cheque into database.
	for _, cheque := range cheques {
		drawer.cdb.writeCheque(payee, drawer.address, cheque, true)
	}
	return cheques, nil
}

// issueCheque creates a cheque for issuing specified amount for payee.
//
// The drawer must have a corresponding lottery as a deposit if it wants
// to issue cheque. This lottery contains several potential redeemers of
// this lottery. The probability that each redeemer can redeem is different,
// so the expected amount of money received by each redeemer is the redemption
// probability multiplied by the lottery amount.
//
// A lottery ticket can be divided into n cheques for payment. Therefore, the
// cheque is paid in a cumulative amount. There is a probability of redemption
// in every cheque issued. The probability of redemption of a later-issued cheque
// needs to be strictly greater than that of the first-issued cheque.
//
// Besides, there is another parameter `commit`. If the commit is true, newly
// generated cheque is persisted immediately. Otherwise external commit operation
// is required.
func (drawer *ChequeDrawer) issueCheque(payee common.Address, lotteryId common.Hash, amount uint64, commit bool) (*Cheque, error) {
	cheque := drawer.cdb.readCheque(payee, drawer.address, lotteryId, true)
	if cheque == nil {
		return nil, errors.New("no cheque found")
	}
	lottery := drawer.cdb.readLottery(drawer.address, cheque.LotteryId)
	if lottery == nil {
		return nil, errors.New("broken db, has cheque but no lottery found")
	}
	// Short circuit if lottery is already expired.
	current := drawer.chain.CurrentHeader().Number.Uint64()
	if lottery.RevealNumber <= current {
		return nil, errors.New("expired lottery")
	}
	// Calculate the total assigned deposit in lottery for the specified payer.
	assigned := lottery.Amount >> (len(cheque.Witness) - 1)

	// Calculate new signed probability range according to new cumulative paid amount
	//
	// Note in the following calculation, it may lose precision.
	// In theory amount/assigned won't be very small. So it's safer to calculate
	// percentage first.
	diff := uint64(math.Ceil(float64(amount) / float64(assigned) * float64(cheque.UpperLimit-cheque.LowerLimit+1)))
	if diff == 0 {
		return nil, errors.New("invalid payment amount")
	}
	// Note it's safe to update cheque here even the commit is false.
	// We store the cheque structure in the cache(instead of pointer).
	// So it's safe to modify the uint64(SignedRange), byte slice pointer
	// (RevealRange) and byte array(Signature) here.
	if cheque.SignedRange == maxSignedRange {
		cheque.SignedRange = cheque.LowerLimit + diff - 1
	} else {
		cheque.SignedRange = cheque.SignedRange + diff
	}
	// Ensure we still have enough deposit to cover payment.
	if cheque.SignedRange > cheque.UpperLimit {
		return nil, ErrNotEnoughDeposit
	}
	// Make the signature for cheque.
	cheque.RevealRange = make([]byte, 4)
	binary.BigEndian.PutUint32(cheque.RevealRange, uint32(cheque.SignedRange))
	if drawer.keySigner != nil {
		// If it's testing, use provided key signer.
		if err := cheque.signWithKey(drawer.keySigner); err != nil {
			return nil, err
		}
	} else {
		// Otherwise, use provided clef as the production-environment signer.
		if err := cheque.sign(drawer.chequeSigner); err != nil {
			return nil, err
		}
	}
	if commit {
		drawer.cdb.writeCheque(payee, drawer.address, cheque, true)
	}
	return cheque, nil
}

// wipeLottery wipes the lottery and associated cheques.
func (drawer *ChequeDrawer) wipeLottery(id common.Hash, tmp bool) {
	drawer.cdb.deleteLottery(drawer.address, id, tmp)
	_, addresses := drawer.cdb.listCheques(
		drawer.address,
		func(addr common.Address, lid common.Hash, cheque *Cheque) bool { return lid == id },
	)
	for _, addr := range addresses {
		drawer.cdb.deleteCheque(drawer.address, addr, id, true)
	}
}

// writeLottery is a wrapper of chequedb function.
func (drawer *ChequeDrawer) writeLottery(l *Lottery) {
	drawer.cdb.writeLottery(drawer.address, l.Id, false, l)
}

// Allowance returns the allowance remaining in the specified lottery that can
// be used to make payments to all included receivers.
func (drawer *ChequeDrawer) Allowance(id common.Hash) map[common.Address]uint64 {
	// Filter all cheques associated with given lottery.
	cheques, addresses := drawer.cdb.listCheques(
		drawer.address,
		func(addr common.Address, lid common.Hash, cheque *Cheque) bool { return lid == id },
	)
	// Short circuit if no cheque found.
	if cheques == nil {
		return nil
	}
	// Short circuit of no corresponding lottery found, it should never happen.
	lottery := drawer.cdb.readLottery(drawer.address, id)
	if lottery == nil {
		return nil
	}
	allowance := make(map[common.Address]uint64)
	for index, cheque := range cheques {
		allowance[addresses[index]] = lottery.balance(addresses[index], cheque)
	}
	return allowance
}

// EstimatedExpiry returns the estimated remaining time(block number) while
// cheques can still be written using this deposit.
func (drawer *ChequeDrawer) EstimatedExpiry(lotteryId common.Hash) uint64 {
	lottery := drawer.cdb.readLottery(drawer.address, lotteryId)
	if lottery == nil {
		return 0
	}
	current := drawer.chain.CurrentHeader()
	if current == nil {
		return 0
	}
	height := current.Number.Uint64()
	if lottery.RevealNumber > height {
		return lottery.RevealNumber - height
	}
	return 0 // The lottery is already expired.
}

// ListLotteries returns all active(not revealed yet) lotteries maintained
// by drawer. It's only used in testing.
func (drawer *ChequeDrawer) ListLotteries() []*Lottery {
	return drawer.cdb.listLotteries(drawer.address, false)
}

// SubscribeLotteryEvent registers a subscription of LotteryEvent.
func (drawer *ChequeDrawer) SubscribeLotteryEvent(ch chan<- []LotteryEvent) event.Subscription {
	return drawer.lmgr.subscribeLotteryEvent(ch)
}

// AverageCost returns the average on-chain transaction cost for create/reset deposit.
// Nil means there is no *recent* deposit record at all.
func (drawer *ChequeDrawer) AverageCost() *big.Float {
	return drawer.cw.average()
}
