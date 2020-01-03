// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	basePriceTC       = time.Hour * 10  // time constant for controlling the base price
	tokenSellMaxRatio = 0.9             // total amount/supply limit ratio over which selling price does not increase further
	tsMinDelay        = time.Second * 5 // minimum recommended delay for sending the next command
	tsMaxBurst        = 16              // maximum commands processed in a row before the recommended delay has elapsed
)

// paymentReceiver processes incoming payments and can be implemented using different
// payment technologies
type paymentReceiver interface {
	info() keyValueList
	receivePayment(from enode.ID, proofOfPayment, oldMeta []byte) (value uint64, newMeta []byte, err error)
	requestPayment(from enode.ID, value uint64, meta []byte) uint64
}

// tokenSale handles client balance deposits, conversion to and from service tokens
// and granting connections and capacity changes through a set of commands called "lespay".
type tokenSale struct {
	lock                              sync.Mutex
	clientPool                        *clientPool
	stopCh                            chan struct{}
	receivers                         map[string]paymentReceiver
	receiverNames                     []string
	basePrice, minBasePrice           float64
	totalTokenLimit, totalTokenAmount func() uint64

	qlock                            sync.Mutex
	sq                               *servingQueue
	sources                          map[string]*cmdSource
	delayFactorZero, delayFactorLast mclock.AbsTime
	tsProcessDelay, tsTargetPeriod   time.Duration
}

// newTokenSale creates a new token sale module instance
func newTokenSale(clientPool *clientPool, minBasePrice float64, talkSpeed int) *tokenSale {
	t := &tokenSale{
		clientPool:       clientPool,
		receivers:        make(map[string]paymentReceiver),
		basePrice:        minBasePrice,
		minBasePrice:     minBasePrice,
		totalTokenLimit:  clientPool.totalTokenLimit,
		totalTokenAmount: clientPool.totalTokenAmount,
		stopCh:           make(chan struct{}),
		sq:               newServingQueue(0, 0),
		sources:          make(map[string]*cmdSource),
		delayFactorZero:  mclock.Now(),
		delayFactorLast:  mclock.Now(),
		tsProcessDelay:   time.Second / time.Duration(talkSpeed),
		tsTargetPeriod:   5 * time.Second / time.Duration(talkSpeed),
	}
	t.sq.setThreads(1)
	go func() {
		cleanupCounter := 0
		for {
			select {
			case <-time.After(time.Second * 10):
				t.lock.Lock()
				cost, ok := t.tokenPrice(1, true)
				if cost > t.basePrice*10 || !ok {
					cost = t.basePrice * 10
				}
				t.basePrice += (cost - t.basePrice) * float64(time.Second*10) / float64(basePriceTC)
				if t.basePrice < minBasePrice {
					t.basePrice = minBasePrice
				}
				t.lock.Unlock()

				cleanupCounter++
				if cleanupCounter == 100 {
					t.sourceMapCleanup()
					cleanupCounter = 0
				}
			case <-t.stopCh:
				return
			}
		}
	}()
	return t
}

type (
	// cmdSource represents a source where lespay commands can come from.
	// It can be either an LES connected peer or a UDP address.
	cmdSource struct {
		ch           chan lespayCmd
		delayUntil   mclock.AbsTime
		burstCounter int
	}
	// lespayCmd represents a single lespay command, including the source it came
	// from and the callback that is going to process the results.
	lespayCmd struct {
		cmd    []byte
		id     enode.ID
		freeID string
		send   func([]byte, uint)
	}
)

// priority returns the processing priority for the next command coming from the given
// source. Commands sent before the previously recommended delay has elapsed have a
// lower priority. It also checks whether the number of commands consecutively sent
// before the delay has elapsed exceeds maxBurst and rejects the command instantly if
// necessary.
func (c *cmdSource) priority() (int64, bool) {
	dt := c.delayUntil - mclock.Now()
	if dt <= 0 {
		c.burstCounter = 0
		return 0, true
	}
	if c.burstCounter >= tsMaxBurst {
		return 0, false
	}
	c.burstCounter++
	return -int64(dt), true
}

// addDelay adds the given amount to the recommended delay
func (c *cmdSource) addDelay(now mclock.AbsTime, delay time.Duration) uint {
	dt := time.Duration(c.delayUntil - now)
	if dt <= 0 {
		dt = 0
	}
	dt += delay
	if dt < tsMinDelay {
		dt = tsMinDelay
	}
	c.delayUntil = now + mclock.AbsTime(dt)
	return uint((dt + time.Second - 1) / time.Second)
}

// delayFactor calculates the amount added to the recommended delay after processing
// a single command
func (t *tokenSale) delayFactor(now mclock.AbsTime) time.Duration {
	if now > t.delayFactorZero {
		t.delayFactorZero = now
	}
	t.delayFactorZero += mclock.AbsTime(t.tsTargetPeriod) + t.delayFactorLast - now
	t.delayFactorLast = now
	if now >= t.delayFactorZero {
		return 0
	} else {
		return time.Duration(t.delayFactorZero-now) / 4
	}
}

// sourceMapCleanup removes unnecessary entries from the command source map
func (t *tokenSale) sourceMapCleanup() {
	t.qlock.Lock()
	defer t.qlock.Unlock()

	now := mclock.Now()
	for src, s := range t.sources {
		if s.delayUntil < now {
			delete(t.sources, src)
		}
	}
}

// queueCommand schedules a lespay command (encapsulated in a lespayCmd) for execution
func (t *tokenSale) queueCommand(src string, cmd lespayCmd) bool {
	t.qlock.Lock()
	defer t.qlock.Unlock()

	s := t.sources[src]
	if s == nil {
		s = &cmdSource{}
		t.sources[src] = s
	}
	if s.ch != nil {
		select {
		case s.ch <- cmd:
			return true
		default:
			return false
		}
	}
	s.ch = make(chan lespayCmd, 16)
	s.ch <- cmd

	go func() {
	loop:
		for {
			select {
			case cmd := <-s.ch:
				t.qlock.Lock()
				pri, ok := s.priority()
				t.qlock.Unlock()
				if ok {
					task := t.sq.newTask(nil, 0, pri)
					if !task.start() {
						break loop
					}
					reply := t.runCommand(cmd.cmd, cmd.id, cmd.freeID)
					t.qlock.Lock()
					now := mclock.Now()
					delay := s.addDelay(now, t.delayFactor(now))
					t.qlock.Unlock()
					cmd.send(reply, delay)
					time.Sleep(t.tsProcessDelay)
					task.done()
				} else {
					cmd.send(nil, 0)
				}
			default:
				break loop
			}
			t.qlock.Lock()
			s.ch = nil
			t.qlock.Unlock()
		}
	}()
	return true
}

// stop stops the token sale module
func (t *tokenSale) stop() {
	close(t.stopCh)
	t.sq.stop()
}

// addReceiver adds a new payment receiver module
func (t *tokenSale) addReceiver(id string, r paymentReceiver) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.receivers[id] = r
	t.receiverNames = append(t.receiverNames, id)
}

// tokenPrice returns the PC units required to buy the specified amount of service
// tokens or the PC units received when selling the given amount of tokens.
// Returns false if not possible.
//
// Note: the price of each token unit depends on the current amount of existing tokens
// and the total token limit, first raising from 0 to basePrice linearly, then tends to
// infinity as tokenAmount approaches tokenLimit.
//
// if 0 <= tokenAmount <= tokenLimit/2:
//   tokenPrice = basePrice*tokenAmount/(tokenLimit/2)
// if tokenLimit/2 <= tokenAmount < tokenLimit:
//   tokenPrice = basePrice*tokenLimit/2/(tokenLimit-tokenAmount)
//
// The price of multiple tokens is calculated as an integral based on the above formula.
func (t *tokenSale) tokenPrice(buySellAmount uint64, buy bool) (float64, bool) {
	tokenLimit := t.totalTokenLimit()
	tokenAmount := t.totalTokenAmount()
	if buy {
		if tokenAmount+buySellAmount >= tokenLimit {
			return 0, false
		}
	} else {
		maxAmount := uint64(float64(tokenLimit) * tokenSellMaxRatio)
		if tokenAmount > maxAmount {
			tokenAmount = maxAmount
		}
		if tokenAmount < buySellAmount {
			buySellAmount = tokenAmount
		}
		tokenAmount -= buySellAmount
	}
	r := float64(tokenAmount) / float64(tokenLimit)
	b := float64(buySellAmount) / float64(tokenLimit)
	var relPrice float64
	if r < 0.5 {
		// first purchased token is in the linear range
		if r+b <= 0.5 {
			// all purchased tokens are in the linear range
			relPrice = b * (r + r + b)
			b = 0
		} else {
			// some purchased tokens are in the 1/x range, calculate linear price
			// update starting point and amount left to buy in the 1/x range
			relPrice = (0.5 - r) * (r + 0.5)
			b = r + b - 0.5
			r = 0.5
		}
	}
	if b > 0 {
		// some purchased tokens are in the 1/x range
		l := 1 - r
		if l < 1e-10 {
			return 0, false
		}
		l = -b / l
		if l < -1+1e-10 {
			return 0, false
		}
		relPrice += -math.Log1p(l) / 2
	}
	return t.basePrice * float64(tokenLimit) * relPrice, true
}

// tokenBuyAmount returns the service token amount currently available for the given
// sum of PC units
func (t *tokenSale) tokenBuyAmount(price float64) uint64 {
	tokenLimit := t.totalTokenLimit()
	tokenAmount := t.totalTokenAmount()
	if tokenLimit <= tokenAmount {
		return 0
	}
	r := float64(tokenAmount) / float64(tokenLimit)
	c := price / (t.basePrice * float64(tokenLimit))
	var relTokens float64
	if r < 0.5 {
		// first purchased token is in the linear range
		relTokens = math.Sqrt(r*r+c) - r
		if r+relTokens <= 0.5 {
			// all purchased tokens are in the linear range, no more to spend
			c = 0
		} else {
			// some purchased tokens are in the 1/x range, calculate linear amount
			// update starting point and available funds left to buy in the 1/x range
			relTokens = 0.5 - r
			c -= (0.5 - r) * (r + 0.5)
			r = 0.5
		}
	}
	if c > 0 {
		relTokens -= math.Expm1(-2*c) * (1 - r)
	}
	return uint64(relTokens * float64(tokenLimit))
}

// tokenSellAmount returns the service token amount that needs to be sold in order
// to receive the given sum of PC units. Returns false if not possible.
func (t *tokenSale) tokenSellAmount(price float64) (uint64, bool) {
	tokenLimit := t.totalTokenLimit()
	tokenAmount := t.totalTokenAmount()
	r := float64(tokenAmount) / float64(tokenLimit)
	if r > tokenSellMaxRatio {
		r = tokenSellMaxRatio
	}
	c := price / (t.basePrice * float64(tokenLimit))
	var relTokens float64
	if r > 0.5 {
		// first sold token is in the 1/x range
		relTokens = math.Expm1(2*c) * (1 - r)
		if r-relTokens >= 0.5 || 1-r < 1e-10 {
			// all sold tokens are in the 1/x range, no more to sell
			c = 0
		} else {
			// some sold tokens are in the linear range, calculate price in 1/x range
			// update starting point and remaining price to sell for in the linear range
			relTokens = r - 0.5
			c -= math.Log1p(relTokens/(1-r)) / 2
			r = 0.5
		}
	}
	if c > 0 {
		// some sold tokens are in the linear range
		if x := r*r - c; x >= 0 {
			relTokens += r - math.Sqrt(x)
		} else {
			return 0, false
		}
	}
	return uint64(relTokens * float64(tokenLimit)), true
}

// connection checks whether it is possible with the current balance levels to establish
// requested connection or capacity change and then stay connected for the given amount
// of time. If it is possible and setCap is also true then the client is activated of the
// capacity change is performed. If not then returns how many tokens are missing and how
// much that would currently cost using the specified payment module(s).
func (t *tokenSale) connection(id enode.ID, freeID string, requestedCapacity uint64, stayConnected time.Duration, paymentModule []string, setCap bool) (availableCapacity, tokenBalance, tokensMissing, pcBalance, pcMissing uint64, paymentRequired []uint64, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	tokensMissing, availableCapacity, err = t.clientPool.setCapacityLocked(id, freeID, requestedCapacity, stayConnected, setCap)
	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value.value(t.clientPool.posExpiration(mclock.Now()))
	var meta tokenSaleMeta
	if err := rlp.DecodeBytes([]byte(pb.meta), &meta); err == nil {
		pcBalance = meta.pcBalance
	}
	if tokensMissing == 0 {
		return
	}
	tokenLimit := t.clientPool.totalTokenLimit()
	tokenAmount := t.clientPool.totalTokenAmount()
	if tokenLimit <= tokenAmount || tokenLimit-tokenAmount <= tokensMissing {
		pcMissing = math.MaxUint64
	} else {
		tokensAvailable := tokenLimit - tokenAmount
		pcr := -math.Log(float64(tokensAvailable-tokensMissing)/float64(tokensAvailable)) * t.basePrice
		if pcr > 0 {
			if pcr > maxBalance {
				pcMissing = math.MaxUint64
			} else {
				pcMissing = uint64(pcr)
				if pcMissing > maxBalance {
					pcMissing = math.MaxUint64
				} else {
					if pcMissing > pcBalance {
						pcMissing -= pcBalance
					} else {
						pcMissing = 0
					}
				}
			}
		}
	}
	if pcMissing == 0 {
		return
	}
	paymentRequired = make([]uint64, len(paymentModule))
	for i, recID := range paymentModule {
		if rec, ok := t.receivers[recID]; !ok || pcMissing == math.MaxUint64 {
			paymentRequired[i] = math.MaxUint64
		} else {
			paymentRequired[i] = rec.requestPayment(id, pcMissing, meta.receiverMeta[recID])
		}
	}
	return
}

// deposit credits a payment on the sender's account using the specified payment module
func (t *tokenSale) deposit(id enode.ID, paymentModule string, proofOfPayment []byte) (pcValue, pcBalance uint64, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	pb := t.clientPool.getPosBalance(id)
	var meta tokenSaleMeta
	if err := rlp.DecodeBytes([]byte(pb.meta), &meta); err == nil {
		pcBalance = meta.pcBalance
	}

	pm := t.receivers[paymentModule]
	if pm == nil {
		return 0, pcBalance, fmt.Errorf("Unknown payment receiver '%s'", paymentModule)
	}
	pcValue, meta.receiverMeta[paymentModule], err = pm.receivePayment(id, proofOfPayment, meta.receiverMeta[paymentModule])
	if err != nil {
		return 0, pcBalance, err
	}
	pcBalance += pcValue
	meta.pcBalance = pcBalance
	metaEnc, _ := rlp.EncodeToBytes(&meta)
	t.clientPool.addBalance(id, 0, string(metaEnc))
	return
}

// buyTokens tries to convert the permanent balance (nominated in the server's preferred
// currency, PC) to service tokens. If spendAll is true then it sells the maxSpend amount
// of PC coins if the received service token amount is at least minReceive. If spendAll is
// false then is buys minReceive amount of tokens if it does not cost more than maxSpend
// amount of PC coins.
// if relative is true then maxSpend and minReceive are specified relative to their current
// balances. In this case maxSpend represents the amount under which the PC balance should
// not go and minReceive represents the amount the service token balance should reach.
// This mode is useful when actual conversion is intended to happen and the sender has to
// retry the command after not receiving a reply previously. In this case the sender cannot
// be sure whether the conversion has already happened or not. If relative is true then it
// is impossible to do a conversion twice. In exchange the sender needs to know its current
// balances (which it probably does if it has made a previous call to just ask the current price).
func (t *tokenSale) buyTokens(id enode.ID, maxSpend, minReceive uint64, relative, spendAll bool) (pcBalance, tokenBalance, spend, receive uint64, success bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value.value(t.clientPool.posExpiration(mclock.Now()))
	var meta tokenSaleMeta
	if err := rlp.DecodeBytes([]byte(pb.meta), &meta); err == nil {
		pcBalance = meta.pcBalance
	}
	if relative {
		if pcBalance > maxSpend {
			maxSpend = pcBalance - maxSpend
		} else {
			maxSpend = 0
		}
		if minReceive > tokenBalance {
			minReceive -= tokenBalance
		} else {
			minReceive = 0
		}
	}

	if maxSpend > pcBalance {
		maxSpend = pcBalance
	}
	if spendAll {
		spend = maxSpend
		receive = t.tokenBuyAmount(float64(spend))
		success = receive >= minReceive
	} else {
		receive = minReceive
		if cost, ok := t.tokenPrice(receive, true); ok {
			spend = uint64(cost) + 1 // ensure that we don't sell small amounts for free
		} else {
			spend = math.MaxUint64
		}
		success = spend <= maxSpend
	}
	if success {
		pcBalance -= spend
		tokenBalance += receive
		meta.pcBalance = pcBalance
		metaEnc, _ := rlp.EncodeToBytes(&meta)
		t.clientPool.addBalance(id, int64(receive), string(metaEnc))
	}
	return
}

// sellTokens tries to convert service tokens to permanent balance (nominated in the server's
// preferred currency, PC). Parameters work similarly to buyTokens.
func (t *tokenSale) sellTokens(id enode.ID, maxSell, minRefund uint64, relative, sellAll bool) (pcBalance, tokenBalance, sell, refund uint64, success bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value.value(t.clientPool.posExpiration(mclock.Now()))
	var meta tokenSaleMeta
	if err := rlp.DecodeBytes([]byte(pb.meta), &meta); err == nil {
		pcBalance = meta.pcBalance
	}
	if relative {
		if pcBalance < minRefund {
			minRefund -= pcBalance
		} else {
			minRefund = 0
		}
		if maxSell < tokenBalance {
			maxSell = tokenBalance - maxSell
		} else {
			maxSell = 0
		}
	}

	if maxSell > tokenBalance {
		maxSell = tokenBalance
	}
	if sellAll {
		sell = maxSell
		if r, ok := t.tokenPrice(sell, false); ok {
			refund = uint64(r)
			success = refund >= minRefund
		}
	} else {
		refund = minRefund
		if s, ok := t.tokenSellAmount(float64(refund)); ok {
			sell = s + 1 // ensure that we don't sell small amounts for free
		} else {
			sell = math.MaxUint64
		}
		success = sell <= maxSell
	}
	if success {
		pcBalance += refund
		tokenBalance -= sell
		meta.pcBalance = pcBalance
		metaEnc, _ := rlp.EncodeToBytes(&meta)
		t.clientPool.addBalance(id, -int64(sell), string(metaEnc))
	}
	return
}

// getBalance returns the current PC balance and service token balance
func (t *tokenSale) getBalance(id enode.ID) (pcBalance, tokenBalance uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value.value(t.clientPool.posExpiration(mclock.Now()))
	var meta tokenSaleMeta
	if err := rlp.DecodeBytes([]byte(pb.meta), &meta); err == nil {
		pcBalance = meta.pcBalance
	}
	return
}

// info returns general information about the server, including version info of the
// lespay command set, supported payment modules and token expiration time constant
func (t *tokenSale) info() (version, compatible uint, info keyValueList, receivers []string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	exp, _ := t.clientPool.getExpirationTCs()
	info = info.add("tokenExpiration", strconv.FormatUint(exp, 10))
	return 1, 1, info, t.receiverNames
}

// receiverInfo returns information about the specified payment receiver(s) if supported
func (t *tokenSale) receiverInfo(receiverIDs []string) []keyValueList {
	t.lock.Lock()
	defer t.lock.Unlock()

	res := make([]keyValueList, len(receiverIDs))
	for i, id := range receiverIDs {
		if rec, ok := t.receivers[id]; ok {
			res[i] = rec.info()
		}
	}
	return res
}

// tokenSaleMeta is the "meta" field used by the lespay token sale module. It is
// attached to token balances and it includes the permanent balance of the client
// nominated in the server's preferred currency and the meta fields provided by
// the used payment receivers.
type tokenSaleMeta struct {
	pcBalance    uint64
	receiverMeta map[string][]byte
}

// receiverMetaEnc is used for easy RLP encoding/decoding
type receiverMetaEnc struct {
	Id   string
	Meta []byte
}

// tokenSaleMetaEnc is used for easy RLP encoding/decoding
type tokenSaleMetaEnc struct {
	Id        string
	Version   uint
	PcBalance uint64
	Receivers []receiverMetaEnc
}

// EncodeRLP implements rlp.Encoder
func (t *tokenSaleMeta) EncodeRLP(w io.Writer) error {
	receivers := make([]receiverMetaEnc, len(t.receiverMeta))
	i := 0
	for id, meta := range t.receiverMeta {
		receivers[i] = receiverMetaEnc{id, meta}
		i++
	}
	return rlp.Encode(w, tokenSaleMetaEnc{
		Id:        "tokenSale",
		Version:   1,
		PcBalance: t.pcBalance,
		Receivers: receivers,
	})
}

// DecodeRLP implements rlp.Decoder
func (t *tokenSaleMeta) DecodeRLP(s *rlp.Stream) error {
	if t.receiverMeta == nil {
		t.receiverMeta = make(map[string][]byte)
	}
	var e tokenSaleMetaEnc
	if err := s.Decode(&e); err != nil {
		return err
	}
	if e.Id != "tokenSale" || e.Version != 1 {
		return fmt.Errorf("Unknown balance meta format '%s' version %d", e.Id, e.Version)
	}
	t.receiverMeta = make(map[string][]byte)
	t.pcBalance = e.PcBalance
	for _, r := range e.Receivers {
		t.receiverMeta[r.Id] = r.Meta
	}
	return nil
}

const (
	tsInfo = iota
	tsReceiverInfo
	tsGetBalance
	tsDeposit
	tsBuyTokens
	tsSellTokens
	tsConnection
)

type (
	tsInfoResults struct {
		Version, Compatible uint
		Info                keyValueList
		Receivers           []string
	}
	tsInfoApiResults struct {
		Version, Compatible uint
		Info                keyValueMapDecoded
		Receivers           []string
	}
	tsReceiverInfoParams     []string
	tsReceiverInfoResults    []keyValueList
	tsReceiverInfoApiResults []keyValueMapDecoded
	tsGetBalanceResults      struct {
		PcBalance, TokenBalance uint64
	}
	tsDepositParams struct {
		PaymentModule  string
		ProofOfPayment []byte
	}
	tsDepositResults struct {
		PcValue, PcBalance uint64
		Err                string
	}
	tsBuyTokensParams struct {
		MaxSpend, MinReceive uint64
		Relative, SpendAll   bool
	}
	tsBuyTokensResults struct {
		PcBalance, TokenBalance, Spend, Receive uint64
		Success                                 bool
	}
	tsSellTokensParams struct {
		MaxSell, MinRefund uint64
		Relative, SellAll  bool
	}
	tsSellTokensResults struct {
		PcBalance, TokenBalance, Sell, Refund uint64
		Success                               bool
	}
	tsConnectionParams struct {
		RequestedCapacity, StayConnected uint64
		PaymentModule                    []string
		SetCap                           bool
	}
	tsConnectionResults struct {
		AvailableCapacity, TokenBalance, TokensMissing, PcBalance, PcMissing uint64
		PaymentRequired                                                      []uint64
		Err                                                                  string
	}
)

// runCommand runs an encoded lespay command and returns the encoded results
func (t *tokenSale) runCommand(cmd []byte, id enode.ID, freeID string) []byte {
	var res []byte
	switch cmd[0] {
	case tsInfo:
		var results tsInfoResults
		if len(cmd) == 1 {
			results.Version, results.Compatible, results.Info, results.Receivers = t.info()
			res, _ = rlp.EncodeToBytes(&results)
		}
	case tsReceiverInfo:
		var (
			params  tsReceiverInfoParams
			results tsReceiverInfoResults
		)
		if err := rlp.DecodeBytes(cmd[1:], &params); err == nil {
			results = t.receiverInfo(params)
			res, _ = rlp.EncodeToBytes(&results)
		}
	case tsGetBalance:
		var results tsGetBalanceResults
		if len(cmd) == 1 {
			results.PcBalance, results.TokenBalance = t.getBalance(id)
			res, _ = rlp.EncodeToBytes(&results)
		}
	case tsDeposit:
		var (
			params  tsDepositParams
			results tsDepositResults
		)
		if err := rlp.DecodeBytes(cmd[1:], &params); err == nil {
			results.PcValue, results.PcBalance, err = t.deposit(id, params.PaymentModule, params.ProofOfPayment)
			if err != nil {
				results.Err = err.Error()
			}
			res, _ = rlp.EncodeToBytes(&results)
		}
	case tsBuyTokens:
		var (
			params  tsBuyTokensParams
			results tsBuyTokensResults
		)
		if err := rlp.DecodeBytes(cmd[1:], &params); err == nil {
			results.PcBalance, results.TokenBalance, results.Spend, results.Receive, results.Success =
				t.buyTokens(id, params.MaxSpend, params.MinReceive, params.Relative, params.SpendAll)
			res, _ = rlp.EncodeToBytes(&results)
		}
	case tsSellTokens:
		var (
			params  tsSellTokensParams
			results tsSellTokensResults
		)
		if err := rlp.DecodeBytes(cmd[1:], &params); err == nil {
			results.PcBalance, results.TokenBalance, results.Sell, results.Refund, results.Success =
				t.sellTokens(id, params.MaxSell, params.MinRefund, params.Relative, params.SellAll)
			res, _ = rlp.EncodeToBytes(&results)
		}
	case tsConnection:
		var (
			params  tsConnectionParams
			results tsConnectionResults
		)
		if err := rlp.DecodeBytes(cmd[1:], &params); err == nil {
			results.AvailableCapacity, results.TokenBalance, results.TokensMissing, results.PcBalance, results.PcMissing, results.PaymentRequired, err =
				t.connection(id, freeID, params.RequestedCapacity, time.Duration(params.StayConnected)*time.Second, params.PaymentModule, params.SetCap)
			if err != nil {
				results.Err = err.Error()
			}
			res, _ = rlp.EncodeToBytes(&results)
		}
	}
	return res
}

type keyValueMapDecoded map[string]interface{}

// DecodeRLP implements rlp.Decoder
func (k *keyValueMapDecoded) DecodeRLP(s *rlp.Stream) error {
	var list keyValueList
	if err := s.Decode(&list); err != nil {
		return err
	}
	*k = make(keyValueMapDecoded)
	for _, item := range list {
		var s string
		if err := rlp.DecodeBytes(item.Value, &s); err != nil {
			return err
		}
		(*k)[item.Key] = s
	}
	return nil
}

// testReceiver implements paymentReceiver. It should only be used for testing.
type testReceiver struct{}

func (t testReceiver) info() keyValueList {
	var info keyValueList
	info = info.add("description", "Test payment receiver")
	info = info.add("version", "1.0.0")
	return info
}

// receivePayment implements paymentReceiver. proofOfPayment is a base 10 ascii number
// which is credited to the sender's account without any further conditions.
func (t testReceiver) receivePayment(from enode.ID, proofOfPayment, oldMeta []byte) (value uint64, newMeta []byte, err error) {
	if len(proofOfPayment) > 8 {
		err = fmt.Errorf("proof of payment is too long; max 8 bytes long big endian integer expected")
		return
	}
	var b [8]byte
	copy(b[8-len(proofOfPayment):], proofOfPayment)
	value = binary.BigEndian.Uint64(b[:])
	return
}

// requestPayment implements paymentReceiver
func (t testReceiver) requestPayment(from enode.ID, value uint64, meta []byte) uint64 {
	return value
}
