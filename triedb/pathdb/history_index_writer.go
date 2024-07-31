// Copyright 2024 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const (
	indexBlockDescSize   = 24   // The size of index block descriptor
	indexBlockEntriesCap = 4096 // The maximum number of entries can be grouped in a block
	indexBlockRestartLen = 256  // The restart interval length of index block

	// stateWriteBatch is the number of states for constructing indexes together.
	// In the worst case, the database write caused by each state is roughly 4KB,
	// 256MB in total is still acceptable.
	stateWriteBatch = 65536
)

// indexBlockDesc is the descriptor of an index block that contains a list of
// state mutation records belonging to a specific state (account or storage slot).
type indexBlockDesc struct {
	min     uint64
	max     uint64
	entries uint32
	id      uint32
}

func newIndexBlockDesc(id uint32) *indexBlockDesc {
	return &indexBlockDesc{id: id}
}

func (d *indexBlockDesc) empty() bool {
	return d.entries == 0
}

func (d *indexBlockDesc) full() bool {
	return d.entries >= indexBlockEntriesCap
}

// encode packs index block descriptor into byte stream.
func (d *indexBlockDesc) encode() []byte {
	var buf [indexBlockDescSize]byte
	binary.BigEndian.PutUint64(buf[:8], d.min)
	binary.BigEndian.PutUint64(buf[8:16], d.max)
	binary.BigEndian.PutUint32(buf[16:20], d.entries)
	binary.BigEndian.PutUint32(buf[20:24], d.id)
	return buf[:]
}

// decode unpacks index block descriptor from byte stream.
func (d *indexBlockDesc) decode(blob []byte) {
	d.min = binary.BigEndian.Uint64(blob[:8])
	d.max = binary.BigEndian.Uint64(blob[8:16])
	d.entries = binary.BigEndian.Uint32(blob[16:20])
	d.id = binary.BigEndian.Uint32(blob[20:24])
}

type blockWriter struct {
	desc     *indexBlockDesc
	restarts []uint32
	scratch  []byte
	buf      []byte
}

func newBlockWriter(blob []byte, desc *indexBlockDesc) (*blockWriter, error) {
	scratch := make([]byte, binary.MaxVarintLen64)
	if len(blob) == 0 {
		return &blockWriter{
			desc:    desc,
			scratch: scratch,
			buf:     make([]byte, 0, 1024),
		}, nil
	}
	restarts, data, err := parseIndexBlock(blob)
	if err != nil {
		return nil, err
	}
	return &blockWriter{
		desc:     desc,
		restarts: restarts,
		scratch:  scratch,
		buf:      data, // safe to own the slice
	}, nil
}

func (b *blockWriter) append(id uint64) error {
	if id <= b.desc.max {
		return fmt.Errorf("element out of order, last: %d, this: %d", b.desc.max, id)
	}
	if b.desc.entries%indexBlockRestartLen == 0 {
		b.restarts = append(b.restarts, uint32(len(b.buf)))

		// The restart point item can be either encoded in variable
		// size or fixed size. Although variable-size encoding is
		// slightly slower (2ns per operation), it is still relatively
		// fast, therefore, it's picked for better space efficiency.
		n := binary.PutUvarint(b.scratch[0:], id)
		b.buf = append(b.buf, b.scratch[:n]...)
	} else {
		n := binary.PutUvarint(b.scratch[0:], id-b.desc.max)
		b.buf = append(b.buf, b.scratch[:n]...)
	}
	b.desc.entries++
	if b.desc.min == 0 {
		b.desc.min = id
	}
	b.desc.max = id
	return nil
}

func (b *blockWriter) empty() bool {
	return b.desc.empty()
}

func (b *blockWriter) full() bool {
	return b.desc.full()
}

func (b *blockWriter) finish() error {
	b.restarts = append(b.restarts, uint32(len(b.restarts)))
	for _, number := range b.restarts {
		binary.BigEndian.PutUint32(b.scratch[:4], number)
		b.buf = append(b.buf, b.scratch[:4]...)
	}
	return nil
}

type indexWriter struct {
	descList []*indexBlockDesc
	last     uint64
	bw       *blockWriter
	frozen   []*blockWriter

	db    ethdb.KeyValueStore
	addr  common.Address
	state common.Hash
}

func newIndexWriter(db ethdb.KeyValueStore, addr common.Address, state common.Hash) (*indexWriter, error) {
	blob := rawdb.ReadStateIndex(db, addr, state)
	if len(blob) == 0 {
		desc := &indexBlockDesc{}
		bw, _ := newBlockWriter(nil, desc)
		return &indexWriter{
			descList: []*indexBlockDesc{desc},
			bw:       bw,
			db:       db,
			addr:     addr,
			state:    state,
		}, nil
	}
	descList, err := parseIndex(blob)
	if err != nil {
		return nil, err
	}
	// Open the last block writer, or create a new one in case
	// it's already full.
	var (
		lastDesc = descList[len(descList)-1]
		lastElem = lastDesc.max
	)
	if lastDesc.full() {
		descList = append(descList, &indexBlockDesc{id: lastDesc.id + 1})
		lastDesc = descList[len(descList)-1]
	}

	indexBlock := rawdb.ReadStateIndexBlock(db, addr, state, lastDesc.id)
	bw, err := newBlockWriter(indexBlock, lastDesc)
	if err != nil {
		return nil, err
	}
	return &indexWriter{
		descList: descList,
		last:     lastElem,
		bw:       bw,
		db:       db,
		addr:     addr,
		state:    state,
	}, nil
}

func (w *indexWriter) append(id uint64) error {
	if id <= w.last {
		return fmt.Errorf("element out of order, last: %d, this: %d", w.last, id)
	}
	if err := w.bw.append(id); err != nil {
		return err
	}
	w.last = id

	if w.bw.full() {
		w.rotate()
	}
	return nil
}

func (w *indexWriter) rotate() {
	w.frozen = append(w.frozen, w.bw)
	desc := newIndexBlockDesc(w.bw.desc.id + 1)
	w.bw, _ = newBlockWriter(nil, desc)
	w.descList = append(w.descList, desc)
}

func (w *indexWriter) finish(batch ethdb.Batch) error {
	var (
		writers  = append(w.frozen, w.bw)
		descList = w.descList
	)
	// Chop the last block if it's empty
	if w.bw.empty() {
		writers = writers[:len(writers)-1]
		descList = descList[:len(descList)-1]
	}
	if len(writers) == 0 {
		return nil
	}
	for _, bw := range writers {
		if err := bw.finish(); err != nil {
			return err
		}
		rawdb.WriteStateIndexBlock(batch, w.addr, w.state, bw.desc.id, bw.buf)
	}
	buf := make([]byte, 0, indexBlockDescSize*len(descList))
	for _, desc := range descList {
		buf = append(buf, desc.encode()...)
	}
	rawdb.WriteStateIndex(batch, w.addr, w.state, buf)
	return nil
}

type historyWriter struct {
	accounts map[common.Address][]uint64
	storages map[common.Address]map[common.Hash][]uint64
	total    int
}

func newHistoryWriter() *historyWriter {
	return &historyWriter{
		accounts: make(map[common.Address][]uint64),
		storages: make(map[common.Address]map[common.Hash][]uint64),
	}
}

func (w *historyWriter) reset() {
	w.total = 0
	w.accounts = make(map[common.Address][]uint64)
	w.storages = make(map[common.Address]map[common.Hash][]uint64)
}

func (w *historyWriter) addAccount(addr common.Address, number uint64) {
	w.total += 1
	w.accounts[addr] = append(w.accounts[addr], number)
}

func (w *historyWriter) addSlot(addr common.Address, hash common.Hash, number uint64) {
	w.total += 1
	if _, ok := w.storages[addr]; !ok {
		w.storages[addr] = make(map[common.Hash][]uint64)
	}
	w.storages[addr][hash] = append(w.storages[addr][hash], number)
}

func (w *historyWriter) finish(db ethdb.KeyValueStore, force bool, head uint64) error {
	if !force && w.total < stateWriteBatch {
		return nil
	}
	var (
		s     = time.Now()
		batch = db.NewBatch()
	)
	for account, idList := range w.accounts {
		iw, err := newIndexWriter(db, account, common.Hash{})
		if err != nil {
			return err
		}
		for _, id := range idList {
			if err := iw.append(id); err != nil {
				return err
			}
		}
		if err := iw.finish(batch); err != nil {
			return err
		}
	}
	for account, slots := range w.storages {
		for slot, idList := range slots {
			iw, err := newIndexWriter(db, account, slot)
			if err != nil {
				return err
			}
			for _, id := range idList {
				if err := iw.append(id); err != nil {
					return err
				}
			}
			if err := iw.finish(batch); err != nil {
				return err
			}
		}
	}
	rawdb.WriteStateHistoryIndexHead(batch, head)
	if err := batch.Write(); err != nil {
		return err
	}
	w.reset()
	log.Info("Written state history indexes", "head", head, "size", common.StorageSize(batch.ValueSize()), "elapsed", common.PrettyDuration(time.Since(s)))
	return nil
}
