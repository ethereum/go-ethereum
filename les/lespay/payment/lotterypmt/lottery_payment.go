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

package lotterypmt

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/lotterybook"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/lespay/payment"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// Identity is the unique string identity of lottery payment.
	Identity = "Lottery"

	// revealPeriod is the full life cycle length of lottery. The
	// number is quite arbitrary here, it's around 6.4 hours. We
	// can set a more reasonable number later.
	revealPeriod = 5760

	// chainSyncedThreshold is the maximum time different that
	// local chain is considered synced. It's around 20 blocks.
	chainSyncedThreshold = time.Minute * 5
)

var ErrNotSynced = errors.New("local chain is not synced")

// chainWatcher is a special helper structure which can determine whether
// the local chain is lag behine or keep synced.
// It's necessary feature for lottery payment seems we have a strong assumption
// that local chain is synced. All contract state we visited is associated
// with chain height. If the local chain is lag behine, these scenarios can
// happen:
// - cheque drawer uses expired lottery for payment
// - cheque drawer will wait very long time for transaction confirmation
//   (finally lead to a timeout error)
// Now this structure is mainly used in the client side. Server has this protocol
// constraint that all client connections will be rejected before finishing sync.
type chainWatcher struct {
	chain  payment.ChainReader
	status uint32
}

func (cw *chainWatcher) run() {
	newHeadCh := make(chan core.ChainHeadEvent, 1024)
	sub := cw.chain.SubscribeChainHeadEvent(newHeadCh)
	if sub == nil {
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-newHeadCh:
			timestamp := time.Unix(int64(ev.Block.Time()), 0)

			// If the time difference is less than 5 minutes(~20 blocks), we
			// can assume the block is latest. But it's also problematic if
			// local machine time is not correct.
			if time.Since(timestamp) < chainSyncedThreshold {
				atomic.StoreUint32(&cw.status, 1)
			} else {
				atomic.StoreUint32(&cw.status, 0)
			}
		case <-sub.Err():
			return
		}
	}
}

// chainSynced returns the indicator whether the local chain is synced.
func (cw *chainWatcher) chainSynced() bool {
	return atomic.LoadUint32(&cw.status) == 1
}

// PaymentSender is the instance can be used to send payment through
// the underlying lottery contract. Usually the sender is a light client
// so that an additional chain watcher is necessary to reject operations
// before the local header chain is synced.
type PaymentSender struct {
	contract    common.Address
	chainReader payment.ChainReader
	sender      *lotterybook.ChequeDrawer
	cwatcher    *chainWatcher
}

func NewPaymentSender(chainReader payment.ChainReader, txSigner *bind.TransactOpts, chequeSigner func(digestHash []byte) ([]byte, error), local, contract common.Address, cBackend bind.ContractBackend, dBackend bind.DeployBackend, db ethdb.Database) (*PaymentSender, error) {
	s := &PaymentSender{
		contract:    contract,
		chainReader: chainReader,
		cwatcher:    &chainWatcher{chain: chainReader},
	}
	sender, err := lotterybook.NewChequeDrawer(local, contract, txSigner, chequeSigner, chainReader, cBackend, dBackend, db)
	if err != nil {
		return nil, err
	}
	s.sender = sender
	go s.cwatcher.run()
	return s, nil
}

func (s *PaymentSender) deposit(receivers []common.Address, amounts []uint64, period uint64, nonce *uint64, gasprice *big.Int) (common.Hash, error) {
	if !s.cwatcher.chainSynced() {
		return common.Hash{}, ErrNotSynced
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFn()

	current := s.chainReader.CurrentHeader().Number.Uint64()
	id, err := s.sender.Deposit(ctx, receivers, amounts, current+period, nonce, gasprice)
	return id, err
}

type CallArgs struct {
	Nonce  *uint64
	Price  *big.Int
	NoWait bool
}

// Deposit creates deposit for the given batch of receivers and corresponding
// deposit amount. If wait is true then a channel is returned, the channel will
// be closed only until the deposit is available for payment and emit a signal
// for it.
//
// The example usage:
//
//    notification, err := sender.Deposit(receivers, amounts, perid, nil)
//    if err != nil {
//        if err == ErrTransactionFailed { // do something }
//        if err == ErrTransactionNotConfirmed { // resend }
//    }
//    event := <-notification
//    if event == nil { // do something }
//    if event.Status = lotterybook.LotteryLost { // resend }
//    if event.Status = lotterybook.LotteryActive { // start to use }
func (s *PaymentSender) Deposit(receivers []common.Address, amounts []uint64, period uint64, callArgs *CallArgs) (chan *lotterybook.LotteryEvent, error) {
	if period == 0 {
		period = revealPeriod
	}
	var (
		nonce *uint64
		price *big.Int
	)
	if callArgs != nil {
		nonce = callArgs.Nonce
		price = callArgs.Price
	}
	id, err := s.deposit(receivers, amounts, period, nonce, price)
	if err != nil {
		return nil, err
	}
	if callArgs != nil && callArgs.NoWait {
		return nil, nil
	}
	done := make(chan *lotterybook.LotteryEvent, 1)
	go func() {
		sink := make(chan []lotterybook.LotteryEvent, 64)
		sub := s.sender.SubscribeLotteryEvent(sink)
		defer sub.Unsubscribe()

		for {
			select {
			case events := <-sink:
				for _, event := range events {
					if event.Id == id {
						done <- &event
						return
					}
				}
			case <-sub.Err():
				done <- nil
				return
			}
		}
	}()
	return done, nil
}

// Pay initiates a payment to the designated payee with specified
// payemnt amount.
func (s *PaymentSender) Pay(payee common.Address, amount uint64) ([][]byte, error) {
	if !s.cwatcher.chainSynced() {
		return nil, ErrNotSynced
	}
	cheques, err := s.sender.IssueCheque(payee, amount)
	if err != nil {
		return nil, err
	}
	proofOfPayments := make([][]byte, len(cheques))
	for index, cheque := range cheques {
		proof, err := rlp.EncodeToBytes(cheque)
		if err != nil {
			return nil, err
		}
		proofOfPayments[index] = proof
	}
	log.Debug("Generated payment", "amount", amount, "payee", payee)
	return proofOfPayments, nil
}

// Destory exits the payment and withdraws all expired lotteries
func (s *PaymentSender) Destory() error {
	if !s.cwatcher.chainSynced() {
		return ErrNotSynced
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFn()
	return s.sender.Destroy(ctx)
}

// Contract returns the contract address used by sender.
func (s *PaymentSender) Contract() common.Address {
	return s.contract
}

// AverageCost returns the average on-chain transaction fee cost in the past day.
func (s *PaymentSender) AverageCost() *big.Float {
	return s.sender.AverageCost()
}

// DebugInspect returns all maintained lotteries inspection(only in testing)
func (s *PaymentSender) DebugInspect() string {
	var msg string
	lotteries := s.sender.ListLotteries()
	for _, l := range lotteries {
		msg += fmt.Sprintf("lottery <%x>: amount: %d, reveal: %d expiration: %d\n", l.Id, l.Amount, l.RevealNumber, s.sender.EstimatedExpiry(l.Id))
		allowance := s.sender.Allowance(l.Id)
		for payee, balance := range allowance {
			msg += fmt.Sprintf("\tpayee %x, balance %d\n", payee, balance)
		}
	}
	return msg
}

// PaymentReceiver is the enter point of the lottery payment receiver
// It defines the function wrapper of the underlying payment methods
// and offers the payment scheme codec.
type PaymentReceiver struct {
	contract common.Address
	receiver *lotterybook.ChequeDrawee
}

// NewPaymentReceiver returns the instance for lottery payment.
// The biggest different between the receiver and sender is:
// usually the receiver is a fullnode which already have the
// protocol constraint that it will reject clients before syncing.
// So chain watcher is not necessary here.
func NewPaymentReceiver(chainReader payment.ChainReader, txSigner *bind.TransactOpts, local, contract common.Address, cBackend bind.ContractBackend, dBackend bind.DeployBackend, db ethdb.Database) (*PaymentReceiver, error) {
	receiver, err := lotterybook.NewChequeDrawee(txSigner, local, contract, chainReader, cBackend, dBackend, db)
	if err != nil {
		return nil, err
	}
	return &PaymentReceiver{
		contract: contract,
		receiver: receiver,
	}, nil
}

// Receive receives a payment from the payer and returns any error
// for payment processing and proving.
func (r *PaymentReceiver) Receive(payer common.Address, proofOfPayment []byte) (uint64, error) {
	var cheque lotterybook.Cheque
	if err := rlp.DecodeBytes(proofOfPayment, &cheque); err != nil {
		return 0, err
	}
	amount, err := r.receiver.AddCheque(payer, &cheque)
	if err != nil {
		return 0, err
	}
	log.Debug("Resolved payment", "amount", amount, "payer", payer)
	return amount, nil
}

// Contract returns the contract address used by receiver.
func (r *PaymentReceiver) Contract() common.Address {
	return r.contract
}

// LotteryPaymentSchema defines the schema of payment.
type LotteryPaymentSchema struct {
	Sender   common.Address
	Receiver common.Address
	Contract common.Address
}

// Identity implements payment.Schema, returns the identity of payment.
func (schema *LotteryPaymentSchema) Identity() string {
	return Identity
}

// Load implements payment.Schema, returns the specified field with given
// entry key.
func (schema *LotteryPaymentSchema) Load(key string) (interface{}, error) {
	typ := reflect.TypeOf(schema).Elem()
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Name == key {
			val := reflect.ValueOf(schema).Elem()
			return val.Field(i).Interface(), nil
		}
	}
	return nil, errors.New("not found")
}

// GenerateSchema returns the payment schema of lottery payment.
func GenerateSchema(contract, local common.Address, sender bool) (payment.SchemaRLP, error) {
	schema := &LotteryPaymentSchema{Contract: contract}
	if sender {
		schema.Sender = local
	} else {
		schema.Receiver = local
	}
	encoded, err := rlp.EncodeToBytes(schema)
	if err != nil {
		return payment.SchemaRLP{}, err
	}
	return payment.SchemaRLP{
		Key:   schema.Identity(),
		Value: encoded,
	}, nil
}

// ResolveSchema resolves the remote schema of lottery payment,
// ensure the schema is compatible with us.
func ResolveSchema(blob []byte, contract common.Address, sender bool) (payment.Schema, error) {
	var schema LotteryPaymentSchema
	if err := rlp.DecodeBytes(blob, &schema); err != nil {
		return nil, err
	}
	// Ensure the contract is compatible
	if schema.Contract != contract {
		return nil, errors.New("imcompatible contract")
	}
	if sender {
		if schema.Receiver == (common.Address{}) {
			return nil, errors.New("empty receiver address")
		}
	} else {
		if schema.Sender == (common.Address{}) {
			return nil, errors.New("empty sender address")
		}
	}
	return &schema, nil
}
