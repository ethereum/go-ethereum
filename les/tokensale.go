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
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	basePriceTC  = time.Hour * 10
	tokenQueueTC = time.Hour
)

type paymentReceiver interface {
	info() keyValueList
	receivePayment(from enode.ID, proofOfPayment, oldMeta []byte) (value uint64, newMeta []byte, err error)
	requestPayment(from enode.ID, value uint64, meta []byte) uint64
}

type tokenSale struct {
	lock, qlock             sync.Mutex
	clientPool              *clientPool
	stopCh                  chan struct{}
	receivers               map[string]paymentReceiver
	receiverNames           []string
	basePrice, minBasePrice float64

	sq      *servingQueue
	sources map[string]*cmdSource
}

func newTokenSale(clientPool *clientPool, minBasePrice float64) *tokenSale {
	t := &tokenSale{
		clientPool:   clientPool,
		receivers:    make(map[string]paymentReceiver),
		basePrice:    minBasePrice,
		minBasePrice: minBasePrice,
		stopCh:       make(chan struct{}),
		sq:           newServingQueue(0, 0),
		sources:      make(map[string]*cmdSource),
	}
	t.sq.setThreads(1)
	go func() {
		cleanupCounter := 0
		for {
			select {
			case <-time.After(time.Second * 10):
				t.lock.Lock()
				cost, ok := t.tokenCost(1)
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
	cmdSource struct {
		ch         chan tokenCmd
		recentTime float64
		lastUpdate mclock.AbsTime
	}

	tokenCmd struct {
		cmd    []byte
		id     enode.ID
		freeID string
		send   func([]byte)
	}
)

func (c *cmdSource) priority(capacity uint64) int64 {
	dt := mclock.Now() - c.lastUpdate
	rt := c.recentTime
	if dt > 0 {
		rt *= math.Exp(-float64(dt) / float64(tokenQueueTC))
	}
	return -int64(rt / float64(capacity))
}

func (c *cmdSource) addTime(time uint64) {
	now := mclock.Now()
	dt := now - c.lastUpdate
	if dt > 0 {
		c.recentTime *= math.Exp(-float64(dt) / float64(tokenQueueTC))
		c.lastUpdate = now
	}
	c.recentTime += float64(time)
}

func (t *tokenSale) sourceMapCleanup() {
	t.qlock.Lock()
	defer t.qlock.Unlock()

	for src, s := range t.sources {
		s.addTime(0)
		if s.recentTime < float64(time.Millisecond*100) {
			delete(t.sources, src)
		}
	}
}

func (t *tokenSale) queueCommand(src string, cmd tokenCmd, capacity uint64) bool {
	t.qlock.Lock()
	defer t.qlock.Unlock()

	s := t.sources[src]
	if s == nil {
		s = &cmdSource{lastUpdate: mclock.Now()}
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
	s.ch = make(chan tokenCmd, 16)
	s.ch <- cmd

	go func() {
	loop:
		for {
			select {
			case cmd := <-s.ch:
				task := t.sq.newTask(nil, 0, s.priority(capacity))
				if !task.start() {
					break loop
				}
				start := mclock.Now()
				reply := t.runCommand(cmd.cmd, cmd.id, cmd.freeID)
				runTime := mclock.Now() - start
				cmd.send(reply)
				time.Sleep(time.Duration(runTime) * 9)
				task.done()
				t.qlock.Lock()
				s.addTime(uint64(runTime))
				t.qlock.Unlock()
			default:
				break loop
			}
			t.qlock.Lock()
			s.ch = nil // TODO map cleanup
			t.qlock.Unlock()
		}
	}()
	return true
}

func (t *tokenSale) stop() {
	close(t.stopCh)
	t.sq.stop()
}

func (t *tokenSale) tokenCost(buyAmount uint64) (float64, bool) {
	tokenLimit := t.clientPool.totalTokenLimit()
	tokenAmount := t.clientPool.totalTokenAmount()
	if tokenAmount+buyAmount >= tokenLimit {
		return 0, false
	}
	r := float64(tokenAmount) / float64(tokenLimit)
	b := float64(buyAmount) / float64(tokenLimit)
	var relCost float64
	if r < 0.5 {
		if r+b <= 0.5 {
			relCost = b * (r + r + b)
			b = 0
		} else {
			relCost = (0.5 - r) * (r + 0.5)
			b = r + b - 0.5
			r = 0.5
		}
	}
	if b > 0 {
		l := 1 - r
		if l < 1e-10 {
			return 0, false
		}
		l = -b / l
		if l < -1+1e-10 {
			return 0, false
		}
		relCost += -math.Log1p(l) / 2

	}
	return t.basePrice * float64(tokenLimit) * relCost, true
}

func (t *tokenSale) tokensFor(maxCost uint64) uint64 {
	tokenLimit := t.clientPool.totalTokenLimit()
	tokenAmount := t.clientPool.totalTokenAmount()
	if tokenLimit <= tokenAmount {
		return 0
	}
	r := float64(tokenAmount) / float64(tokenLimit)
	c := float64(maxCost) / (t.basePrice * float64(tokenLimit))
	var relTokens float64
	if r < 0.5 {
		relTokens = math.Sqrt(r*r+c) - r
		if r+relTokens <= 0.5 {
			c = 0
		} else {
			relTokens = 0.5 - r
			c -= (0.5 - r) * (r + 0.5)
		}
	}
	if c > 0 {
		relTokens += -math.Expm1(-2*c) * (1 - r)
	}
	return uint64(relTokens * float64(tokenLimit))
}

func (t *tokenSale) connection(id enode.ID, freeID string, requestedCapacity uint64, stayConnected time.Duration, paymentModule []string, setCap bool) (availableCapacity, tokenBalance, tokensMissing, pcBalance, pcMissing uint64, paymentRequired []uint64, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	tokensMissing, availableCapacity, err = t.clientPool.setCapacityLocked(id, freeID, requestedCapacity, stayConnected, setCap)
	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value
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

func (t *tokenSale) buyTokens(id enode.ID, maxSpend, minReceive uint64, relative, spendAll bool) (pcBalance, tokenBalance, spend, receive uint64, success bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value
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
		receive = t.tokensFor(spend)
		success = receive >= minReceive
	} else {
		receive = minReceive
		if cost, ok := t.tokenCost(receive); ok {
			spend = uint64(cost)
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

func (t *tokenSale) getBalance(id enode.ID) (pcBalance, tokenBalance uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	pb := t.clientPool.getPosBalance(id)
	tokenBalance = pb.value
	var meta tokenSaleMeta
	if err := rlp.DecodeBytes([]byte(pb.meta), &meta); err == nil {
		pcBalance = meta.pcBalance
	}
	return
}

func (t *tokenSale) info() (version, compatible uint, info keyValueList, receivers []string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return 1, 1, keyValueList{}, t.receiverNames
}

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

type tokenSaleMeta struct {
	pcBalance    uint64
	receiverMeta map[string][]byte
}

type receiverMetaEnc struct {
	Id   string
	Meta []byte
}

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
	tsConnection
)

type (
	tsInfoResults struct {
		Version, Compatible uint
		Info                keyValueList
		Receivers           []string
	}
	tsReceiverInfoParams  []string
	tsReceiverInfoResults []keyValueList
	tsGetBalanceResults   struct {
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
