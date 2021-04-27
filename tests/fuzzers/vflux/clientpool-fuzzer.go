// Copyright 2021 The go-ethereum Authors
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

package vflux

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"io"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/les/vflux"
	vfs "github.com/ethereum/go-ethereum/les/vflux/server"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

func init() {
	utils.Fuzzing = true
}

type fuzzer struct {
	peers                  []*clientPeer
	peerCount              int
	disconnectList         []*clientPeer
	input                  io.Reader
	exhausted              bool
	activeCount, activeCap uint64
	maxCount, maxCap       uint64
	serialNumber           uint64
}

type clientPeer struct {
	fuzzer  *fuzzer
	node    *enode.Node
	freeID  string
	privKey *ecdsa.PrivateKey
	address []byte
	timeout time.Duration

	balance        vfs.ConnectedBalance
	capacity       uint64
	totalDeposited *big.Int
}

func (p *clientPeer) Node() *enode.Node {
	return p.node
}

func (p *clientPeer) FreeClientId() string {
	return p.freeID
}

func (p *clientPeer) InactiveAllowance() time.Duration {
	return p.timeout
}

func (p *clientPeer) UpdateCapacity(newCap uint64, requested bool) {
	p.fuzzer.activeCap -= p.capacity
	if p.capacity != 0 {
		p.fuzzer.activeCount--
	}
	p.capacity = newCap
	p.fuzzer.activeCap += p.capacity
	if p.capacity != 0 {
		p.fuzzer.activeCount++
	}
}

func (p *clientPeer) Disconnect() {
	p.fuzzer.disconnectList = append(p.fuzzer.disconnectList, p)
	p.fuzzer.activeCap -= p.capacity
	if p.capacity != 0 {
		p.fuzzer.activeCount--
	}
	p.capacity = 0
	p.balance = nil
}

var testKeys = []string{
	"b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291",
	"8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a",
	"49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee",
	"45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8",
	"eef77acb6c6a6eebc5b363a475ac583ec7eccdb42b6481424c60f59aa326547f",
	"66fb62bfbd66b9177a138c1e5cddbe4f7c30c343e94e68df8769459cb1cde628",
	"0288ef00023598499cb6c940146d050d2b1fb914198c327f76aad590bead68b6",
	"869d6ecf5211f1cc60418a13b9d870b22959d0c16f02bec714c960dd2298a32d",
	"e238eb8e04fee6511ab04c6dd3c89ce097b11f25d584863ac2b6d5b35b1847e4",
}

func newFuzzer(input []byte) *fuzzer {
	f := &fuzzer{
		input: bytes.NewReader(input),
	}
	f.peerCount = 1 << f.randomInt(9)
	f.peers = make([]*clientPeer, f.peerCount)
	//randReader := rand.New(rand.NewSource(42)) // deterministic pseudo-random reader

	for i := range f.peers {
		testKey, _ := crypto.HexToECDSA(testKeys[i%len(testKeys)])
		f.peers[i] = &clientPeer{
			fuzzer:         f,
			node:           enode.SignNull(new(enr.Record), enode.ID{byte(i)}),
			freeID:         string([]byte{byte(i)}),
			privKey:        testKey,
			address:        crypto.PubkeyToAddress(testKey.PublicKey).Bytes(),
			timeout:        f.randomDelay(),
			totalDeposited: big.NewInt(0),
		}
	}
	return f
}

func (f *fuzzer) read(size int) []byte {
	out := make([]byte, size)
	if _, err := f.input.Read(out); err != nil {
		f.exhausted = true
	}
	return out
}

func (f *fuzzer) randomByte() byte {
	d := f.read(1)
	return d[0]
}

func (f *fuzzer) randomBool() bool {
	d := f.read(1)
	return d[0]&1 == 1
}

func (f *fuzzer) randomInt(max int) int {
	if max == 0 {
		return 0
	}
	if max <= 256 {
		return int(f.randomByte()) % max
	}
	var a uint16
	if err := binary.Read(f.input, binary.LittleEndian, &a); err != nil {
		f.exhausted = true
	}
	return int(a % uint16(max))
}

func (f *fuzzer) randomPeer() *clientPeer {
	return f.peers[f.randomInt(f.peerCount)]
}

func (f *fuzzer) randomTokenAmount(signed bool) int64 {
	x := uint64(f.randomInt(65000))
	x = x * x * x * x

	if signed && (x&1) == 1 {
		if x <= math.MaxInt64 {
			return -int64(x)
		}
		return math.MinInt64
	}
	if x <= math.MaxInt64 {
		return int64(x)
	}
	return math.MaxInt64
}

func (f *fuzzer) randomDelay() time.Duration {
	delay := f.randomByte()
	if delay < 128 {
		return time.Duration(delay) * time.Second
	}
	return 0
}

func (f *fuzzer) randomFactors() vfs.PriceFactors {
	return vfs.PriceFactors{
		TimeFactor:     float64(f.randomByte()) / 25500,
		CapacityFactor: float64(f.randomByte()) / 255,
		RequestFactor:  float64(f.randomByte()) / 255,
	}
}

func (f *fuzzer) connectedBalanceOp(balance vfs.ConnectedBalance) {
	switch f.randomInt(3) {
	case 0:
		balance.RequestServed(uint64(f.randomTokenAmount(false)))
	case 1:
		balance.SetPriceFactors(f.randomFactors(), f.randomFactors())
	case 2:
		balance.GetBalance()
		balance.GetRawBalance()
		balance.GetPriceFactors()
	}
}

func (f *fuzzer) atomicBalanceOp(balance vfs.AtomicBalanceOperator) {
	switch f.randomInt(3) {
	case 0:
		balance.AddBalance(f.randomTokenAmount(true))
	case 1:
		balance.SetBalance(uint64(f.randomTokenAmount(false)), uint64(f.randomTokenAmount(false)))
	case 2:
		balance.GetBalance()
		balance.GetRawBalance()
		balance.GetPriceFactors()
	}
}

func (f *fuzzer) randomIntOrInf() vflux.IntOrInf {
	v := vflux.IntOrInf{Type: uint8(f.randomInt(4))}
	if v.Type < 2 {
		v.Value = *big.NewInt(f.randomTokenAmount(false))
	}
	return v
}

func (f *fuzzer) randomVfxReq(peer *clientPeer) (string, []byte) {
	switch f.randomInt(5) {
	case 0:
		req := vflux.CapacityQueryRequest{
			Bias:      uint64(f.randomByte()),
			AddTokens: make([]vflux.IntOrInf, f.randomInt(vflux.CapacityQueryMaxLen+1)),
		}
		for i := range req.AddTokens {
			req.AddTokens[i] = f.randomIntOrInf()
		}
		reqEnc, err := rlp.EncodeToBytes(&req)
		if err != nil {
			panic(err)
		}
		return vflux.CapacityQueryName, reqEnc

	case 1:
		f.serialNumber++
		req := vflux.ExchangeRequest{
			SerialNumber:   f.serialNumber,
			CurrencyId:     "eth",
			PaymentAddress: peer.address,
			MinTokens:      f.randomIntOrInf(),
			MaxTokens:      f.randomIntOrInf(),
			MaxCurrency:    f.randomIntOrInf(),
		}
		req.Sign(peer.node.ID(), peer.privKey)
		reqEnc, err := rlp.EncodeToBytes(&req)
		if err != nil {
			panic(err)
		}
		return vflux.ExchangeName, reqEnc

	case 2:
		req := vflux.PriceQueryRequest{
			CurrencyId:   "eth",
			TokenAmounts: make([]vflux.IntOrInf, f.randomInt(vflux.PriceQueryMaxLen+1)),
		}
		for i := range req.TokenAmounts {
			req.TokenAmounts[i] = f.randomIntOrInf()
		}
		reqEnc, err := rlp.EncodeToBytes(&req)
		if err != nil {
			panic(err)
		}
		return vflux.PriceQueryName, reqEnc

	case 3:
		peer.totalDeposited.Add(peer.totalDeposited, big.NewInt(f.randomTokenAmount(false)))
		req := vflux.DepositRequest{
			CurrencyId:      "eth",
			PaymentAddress:  peer.address,
			PaymentReceiver: "dummy",
			PaymentData:     peer.totalDeposited.Bytes(),
		}
		reqEnc, err := rlp.EncodeToBytes(&req)
		if err != nil {
			panic(err)
		}
		return vflux.DepositName, reqEnc

	case 4:
		req := vflux.GetBalanceRequest{
			CurrencyId:     "eth",
			PaymentAddress: peer.address,
		}
		req.Sign(peer.node.ID(), peer.privKey)
		reqEnc, err := rlp.EncodeToBytes(&req)
		if err != nil {
			panic(err)
		}
		return vflux.GetBalanceName, reqEnc
	}
	return "", nil
}

func FuzzClientPool(input []byte) int {
	if len(input) > 10000 {
		return -1
	}
	f := newFuzzer(input)
	if f.exhausted {
		return 0
	}
	clock := &mclock.Simulated{}
	db := memorydb.New()
	pool := vfs.NewClientPool(db, 10, f.randomDelay(), clock, func() bool { return true })
	pool.AddTokenSale("eth", 1e-6)
	pool.AddPaymentReceiver("dummy", vfs.NewDummyPaymentReceiver(db, []byte("dummyPayment:")))
	pool.Start()
	defer pool.Stop()

	count := 0
	for !f.exhausted && count < 1000 {
		count++
		switch f.randomInt(11) {
		case 0:
			p := f.randomPeer()
			p.balance = pool.Register(p)
		case 1:
			f.randomPeer().Disconnect()
		case 2:
			f.maxCount = uint64(f.randomByte())
			f.maxCap = uint64(f.randomByte())
			f.maxCap *= f.maxCap
			pool.SetLimits(f.maxCount, f.maxCap)
		case 3:
			pool.SetConnectedBias(f.randomDelay())
		case 4:
			pool.SetDefaultFactors(f.randomFactors(), f.randomFactors())
		case 5:
			pool.SetExpirationTCs(uint64(f.randomInt(50000)), uint64(f.randomInt(50000)))
		case 6:
			if _, err := pool.SetCapacity(f.randomPeer().node, uint64(f.randomByte()), f.randomDelay(), f.randomBool()); err == vfs.ErrCantFindMaximum {
				panic(nil)
			}
		case 7:
			if balance := f.randomPeer().balance; balance != nil {
				f.connectedBalanceOp(balance)
			}
		case 8:
			pool.BalanceOperation(f.randomPeer().node.ID(), f.randomPeer().freeID, nil, func(balance vfs.AtomicBalanceOperator) {
				count := f.randomInt(4)
				for i := 0; i < count; i++ {
					f.atomicBalanceOp(balance)
				}
			})
		case 9:
			pool.TotalTokenAmount()
			pool.GetExpirationTCs()
			pool.Active()
			pool.Limits()
			pool.GetPosBalanceIDs(f.randomPeer().node.ID(), f.randomPeer().node.ID(), f.randomInt(100))
		case 10:
			peer, freeID := f.randomPeer(), f.randomPeer().freeID
			for i := f.randomInt(8); i > 0; i-- {
				name, data := f.randomVfxReq(peer)
				p := int(f.randomByte())
				if p < len(data) {
					data[p] = f.randomByte()
				}
				pool.Handle(peer.node.ID(), freeID, name, data)
			}
		}

		for _, peer := range f.disconnectList {
			pool.Unregister(peer)
		}
		f.disconnectList = nil
		if d := f.randomDelay(); d > 0 {
			clock.Run(d)
		}
		if activeCount, activeCap := pool.Active(); activeCount != f.activeCount || activeCap != f.activeCap {
			panic(nil)
		}
		if f.activeCount > f.maxCount || f.activeCap > f.maxCap {
			panic(nil)
		}
	}
	return 0
}
