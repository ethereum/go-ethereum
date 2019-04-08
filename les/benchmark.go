// Copyright 2018 The go-ethereum Authors
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
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// requestBenchmark is an interface for different randomized request generators
type requestBenchmark interface {
	// init initializes the generator for generating the given number of randomized requests
	init(pm *ProtocolManager, count int) error
	// request initiates sending a single request to the given peer
	request(peer *peer, index int) error
}

// benchmarkBlockHeaders implements requestBenchmark
type benchmarkBlockHeaders struct {
	amount, skip    int
	reverse, byHash bool
	offset, randMax int64
	hashes          []common.Hash
}

func (b *benchmarkBlockHeaders) init(pm *ProtocolManager, count int) error {
	d := int64(b.amount-1) * int64(b.skip+1)
	b.offset = 0
	b.randMax = pm.blockchain.CurrentHeader().Number.Int64() + 1 - d
	if b.randMax < 0 {
		return fmt.Errorf("chain is too short")
	}
	if b.reverse {
		b.offset = d
	}
	if b.byHash {
		b.hashes = make([]common.Hash, count)
		for i := range b.hashes {
			b.hashes[i] = rawdb.ReadCanonicalHash(pm.chainDb, uint64(b.offset+rand.Int63n(b.randMax)))
		}
	}
	return nil
}

func (b *benchmarkBlockHeaders) request(peer *peer, index int) error {
	if b.byHash {
		return peer.RequestHeadersByHash(0, 0, b.hashes[index], b.amount, b.skip, b.reverse)
	} else {
		return peer.RequestHeadersByNumber(0, 0, uint64(b.offset+rand.Int63n(b.randMax)), b.amount, b.skip, b.reverse)
	}
}

// benchmarkBodiesOrReceipts implements requestBenchmark
type benchmarkBodiesOrReceipts struct {
	receipts bool
	hashes   []common.Hash
}

func (b *benchmarkBodiesOrReceipts) init(pm *ProtocolManager, count int) error {
	randMax := pm.blockchain.CurrentHeader().Number.Int64() + 1
	b.hashes = make([]common.Hash, count)
	for i := range b.hashes {
		b.hashes[i] = rawdb.ReadCanonicalHash(pm.chainDb, uint64(rand.Int63n(randMax)))
	}
	return nil
}

func (b *benchmarkBodiesOrReceipts) request(peer *peer, index int) error {
	if b.receipts {
		return peer.RequestReceipts(0, 0, []common.Hash{b.hashes[index]})
	} else {
		return peer.RequestBodies(0, 0, []common.Hash{b.hashes[index]})
	}
}

// benchmarkProofsOrCode implements requestBenchmark
type benchmarkProofsOrCode struct {
	code     bool
	headHash common.Hash
}

func (b *benchmarkProofsOrCode) init(pm *ProtocolManager, count int) error {
	b.headHash = pm.blockchain.CurrentHeader().Hash()
	return nil
}

func (b *benchmarkProofsOrCode) request(peer *peer, index int) error {
	key := make([]byte, 32)
	rand.Read(key)
	if b.code {
		return peer.RequestCode(0, 0, []CodeReq{{BHash: b.headHash, AccKey: key}})
	} else {
		return peer.RequestProofs(0, 0, []ProofReq{{BHash: b.headHash, Key: key}})
	}
}

// benchmarkHelperTrie implements requestBenchmark
type benchmarkHelperTrie struct {
	bloom                 bool
	reqCount              int
	sectionCount, headNum uint64
}

func (b *benchmarkHelperTrie) init(pm *ProtocolManager, count int) error {
	if b.bloom {
		b.sectionCount, b.headNum, _ = pm.server.bloomTrieIndexer.Sections()
	} else {
		b.sectionCount, _, _ = pm.server.chtIndexer.Sections()
		b.headNum = b.sectionCount*params.CHTFrequency - 1
	}
	if b.sectionCount == 0 {
		return fmt.Errorf("no processed sections available")
	}
	return nil
}

func (b *benchmarkHelperTrie) request(peer *peer, index int) error {
	reqs := make([]HelperTrieReq, b.reqCount)

	if b.bloom {
		bitIdx := uint16(rand.Intn(2048))
		for i := range reqs {
			key := make([]byte, 10)
			binary.BigEndian.PutUint16(key[:2], bitIdx)
			binary.BigEndian.PutUint64(key[2:], uint64(rand.Int63n(int64(b.sectionCount))))
			reqs[i] = HelperTrieReq{Type: htBloomBits, TrieIdx: b.sectionCount - 1, Key: key}
		}
	} else {
		for i := range reqs {
			key := make([]byte, 8)
			binary.BigEndian.PutUint64(key[:], uint64(rand.Int63n(int64(b.headNum))))
			reqs[i] = HelperTrieReq{Type: htCanonical, TrieIdx: b.sectionCount - 1, Key: key, AuxReq: auxHeader}
		}
	}

	return peer.RequestHelperTrieProofs(0, 0, reqs)
}

// benchmarkTxSend implements requestBenchmark
type benchmarkTxSend struct {
	txs types.Transactions
}

func (b *benchmarkTxSend) init(pm *ProtocolManager, count int) error {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.NewEIP155Signer(big.NewInt(18))
	b.txs = make(types.Transactions, count)

	for i := range b.txs {
		data := make([]byte, txSizeCostLimit)
		rand.Read(data)
		tx, err := types.SignTx(types.NewTransaction(0, addr, new(big.Int), 0, new(big.Int), data), signer, key)
		if err != nil {
			panic(err)
		}
		b.txs[i] = tx
	}
	return nil
}

func (b *benchmarkTxSend) request(peer *peer, index int) error {
	enc, _ := rlp.EncodeToBytes(types.Transactions{b.txs[index]})
	return peer.SendTxs(0, 0, enc)
}

// benchmarkTxStatus implements requestBenchmark
type benchmarkTxStatus struct{}

func (b *benchmarkTxStatus) init(pm *ProtocolManager, count int) error {
	return nil
}

func (b *benchmarkTxStatus) request(peer *peer, index int) error {
	var hash common.Hash
	rand.Read(hash[:])
	return peer.RequestTxStatus(0, 0, []common.Hash{hash})
}

// benchmarkSetup stores measurement data for a single benchmark type
type benchmarkSetup struct {
	req                   requestBenchmark
	totalCount            int
	totalTime, avgTime    time.Duration
	maxInSize, maxOutSize uint32
	err                   error
}

// runBenchmark runs a benchmark cycle for all benchmark types in the specified
// number of passes
func (pm *ProtocolManager) runBenchmark(benchmarks []requestBenchmark, passCount int, targetTime time.Duration) []*benchmarkSetup {
	setup := make([]*benchmarkSetup, len(benchmarks))
	for i, b := range benchmarks {
		setup[i] = &benchmarkSetup{req: b}
	}
	for i := 0; i < passCount; i++ {
		log.Info("Running benchmark", "pass", i+1, "total", passCount)
		todo := make([]*benchmarkSetup, len(benchmarks))
		copy(todo, setup)
		for len(todo) > 0 {
			// select a random element
			index := rand.Intn(len(todo))
			next := todo[index]
			todo[index] = todo[len(todo)-1]
			todo = todo[:len(todo)-1]

			if next.err == nil {
				// calculate request count
				count := 50
				if next.totalTime > 0 {
					count = int(uint64(next.totalCount) * uint64(targetTime) / uint64(next.totalTime))
				}
				if err := pm.measure(next, count); err != nil {
					next.err = err
				}
			}
		}
	}
	log.Info("Benchmark completed")

	for _, s := range setup {
		if s.err == nil {
			s.avgTime = s.totalTime / time.Duration(s.totalCount)
		}
	}
	return setup
}

// meteredPipe implements p2p.MsgReadWriter and remembers the largest single
// message size sent through the pipe
type meteredPipe struct {
	rw      p2p.MsgReadWriter
	maxSize uint32
}

func (m *meteredPipe) ReadMsg() (p2p.Msg, error) {
	return m.rw.ReadMsg()
}

func (m *meteredPipe) WriteMsg(msg p2p.Msg) error {
	if msg.Size > m.maxSize {
		m.maxSize = msg.Size
	}
	return m.rw.WriteMsg(msg)
}

// measure runs a benchmark for a single type in a single pass, with the given
// number of requests
func (pm *ProtocolManager) measure(setup *benchmarkSetup, count int) error {
	clientPipe, serverPipe := p2p.MsgPipe()
	clientMeteredPipe := &meteredPipe{rw: clientPipe}
	serverMeteredPipe := &meteredPipe{rw: serverPipe}
	var id enode.ID
	rand.Read(id[:])
	clientPeer := pm.newPeer(lpv2, NetworkId, p2p.NewPeer(id, "client", nil), clientMeteredPipe)
	serverPeer := pm.newPeer(lpv2, NetworkId, p2p.NewPeer(id, "server", nil), serverMeteredPipe)
	serverPeer.sendQueue = newExecQueue(count)
	serverPeer.announceType = announceTypeNone
	serverPeer.fcCosts = make(requestCostTable)
	c := &requestCosts{}
	for code := range requests {
		serverPeer.fcCosts[code] = c
	}
	serverPeer.fcParams = flowcontrol.ServerParams{BufLimit: 1, MinRecharge: 1}
	serverPeer.fcClient = flowcontrol.NewClientNode(pm.server.fcManager, serverPeer.fcParams)
	defer serverPeer.fcClient.Disconnect()

	if err := setup.req.init(pm, count); err != nil {
		return err
	}

	errCh := make(chan error, 10)
	start := mclock.Now()

	go func() {
		for i := 0; i < count; i++ {
			if err := setup.req.request(clientPeer, i); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		for i := 0; i < count; i++ {
			if err := pm.handleMsg(serverPeer); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		for i := 0; i < count; i++ {
			msg, err := clientPipe.ReadMsg()
			if err != nil {
				errCh <- err
				return
			}
			var i interface{}
			msg.Decode(&i)
		}
		// at this point we can be sure that the other two
		// goroutines finished successfully too
		close(errCh)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-pm.quitSync:
		clientPipe.Close()
		serverPipe.Close()
		return fmt.Errorf("Benchmark cancelled")
	}

	setup.totalTime += time.Duration(mclock.Now() - start)
	setup.totalCount += count
	setup.maxInSize = clientMeteredPipe.maxSize
	setup.maxOutSize = serverMeteredPipe.maxSize
	clientPipe.Close()
	serverPipe.Close()
	return nil
}
