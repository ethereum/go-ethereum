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
	"encoding/binary"
	"io"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/les/vflux"
	vfs "github.com/ethereum/go-ethereum/les/vflux/server"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

type fuzzer struct {
	peers                  [256]*clientPeer
	disconnectList         []*clientPeer
	input                  io.Reader
	exhausted              bool
	activeCount, activeCap uint64
	maxCount, maxCap       uint64
}

type clientPeer struct {
	fuzzer  *fuzzer
	node    *enode.Node
	freeID  string
	timeout time.Duration

	balance  vfs.ConnectedBalance
	capacity uint64
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

func newFuzzer(input []byte) *fuzzer {
	f := &fuzzer{
		input: bytes.NewReader(input),
	}
	for i := range f.peers {
		f.peers[i] = &clientPeer{
			fuzzer:  f,
			node:    enode.SignNull(new(enr.Record), enode.ID{byte(i)}),
			freeID:  string([]byte{byte(i)}),
			timeout: f.randomDelay(),
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
	pool.Start()
	defer pool.Stop()

	count := 0
	for !f.exhausted && count < 1000 {
		count++
		switch f.randomInt(11) {
		case 0:
			i := int(f.randomByte())
			f.peers[i].balance = pool.Register(f.peers[i])
		case 1:
			i := int(f.randomByte())
			f.peers[i].Disconnect()
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
			if _, err := pool.SetCapacity(f.peers[f.randomByte()].node, uint64(f.randomByte()), f.randomDelay(), f.randomBool()); err == vfs.ErrCantFindMaximum {
				panic(nil)
			}
		case 7:
			if balance := f.peers[f.randomByte()].balance; balance != nil {
				f.connectedBalanceOp(balance)
			}
		case 8:
			pool.BalanceOperation(f.peers[f.randomByte()].node.ID(), f.peers[f.randomByte()].freeID, func(balance vfs.AtomicBalanceOperator) {
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
			pool.GetPosBalanceIDs(f.peers[f.randomByte()].node.ID(), f.peers[f.randomByte()].node.ID(), f.randomInt(100))
		case 10:
			req := vflux.CapacityQueryReq{
				Bias:      uint64(f.randomByte()),
				AddTokens: make([]vflux.IntOrInf, f.randomInt(vflux.CapacityQueryMaxLen+1)),
			}
			for i := range req.AddTokens {
				v := vflux.IntOrInf{Type: uint8(f.randomInt(4))}
				if v.Type < 2 {
					v.Value = *big.NewInt(f.randomTokenAmount(false))
				}
				req.AddTokens[i] = v
			}
			reqEnc, err := rlp.EncodeToBytes(&req)
			if err != nil {
				panic(err)
			}
			p := int(f.randomByte())
			if p < len(reqEnc) {
				reqEnc[p] = f.randomByte()
			}
			pool.Handle(f.peers[f.randomByte()].node.ID(), f.peers[f.randomByte()].freeID, vflux.CapacityQueryName, reqEnc)
		}

		for _, peer := range f.disconnectList {
			pool.Unregister(peer)
		}
		f.disconnectList = nil
		if d := f.randomDelay(); d > 0 {
			clock.Run(d)
		}
		//fmt.Println(f.activeCount, f.maxCount, f.activeCap, f.maxCap)
		if activeCount, activeCap := pool.Active(); activeCount != f.activeCount || activeCap != f.activeCap {
			panic(nil)
		}
		if f.activeCount > f.maxCount || f.activeCap > f.maxCap {
			panic(nil)
		}
	}
	return 0
}
