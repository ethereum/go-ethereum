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
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/merkletree"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

type testEnv struct {
	draweeDb     ethdb.Database
	drawerDb     ethdb.Database
	draweeKey    *ecdsa.PrivateKey
	draweeAddr   common.Address
	drawerKey    *ecdsa.PrivateKey
	drawerAddr   common.Address
	contractAddr common.Address
	backend      *backends.SimulatedBackend
}

func newTestEnv(t *testing.T) *testEnv {
	db1, db2 := rawdb.NewMemoryDatabase(), rawdb.NewMemoryDatabase()
	key, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()
	key3, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	addr2 := crypto.PubkeyToAddress(key2.PublicKey)
	addr3 := crypto.PubkeyToAddress(key3.PublicKey)

	sim := backends.NewSimulatedBackend(core.GenesisAlloc{
		addr:  {Balance: big.NewInt(2e18)},
		addr2: {Balance: big.NewInt(2e18)},
		addr3: {Balance: big.NewInt(2e18)},
	}, 10000000)
	contractAddr, _, _ := DeployLotteryBook(bind.NewKeyedTransactor(key3), sim)
	return &testEnv{
		draweeDb:     db1,
		draweeKey:    key,
		draweeAddr:   addr,
		drawerDb:     db2,
		drawerKey:    key2,
		drawerAddr:   addr2,
		contractAddr: contractAddr,
		backend:      sim,
	}
}

func (env *testEnv) close() { env.backend.Close() }

func (env *testEnv) commitEmptyBlocks(number int) {
	for i := 0; i < number; i++ {
		env.backend.Commit()
	}
}

func (env *testEnv) commitEmptyUntil(end uint64) {
	for {
		if env.backend.Blockchain().CurrentHeader().Number.Uint64() == end {
			return
		}
		env.backend.Commit()
	}
}

func (env *testEnv) checkEvent(sink chan []LotteryEvent, expect []LotteryEvent) bool {
	select {
	case ev := <-sink:
		if len(ev) != len(expect) {
			return false
		}
		for index := range ev {
			if ev[index].Id != expect[index].Id {
				return false
			}
			if ev[index].Status != expect[index].Status {
				return false
			}
		}
	case <-time.NewTimer(time.Second).C:
		fmt.Println("no event")
		return false
	}
	select {
	case <-sink:
		fmt.Println("unexpected event")
		return false // Unexpect incoming events
	case <-time.NewTimer(time.Microsecond * 100).C:
		return true
	}
}

func (env *testEnv) newRawLottery(payees []common.Address, weight []uint64, revealShift uint64) (*Lottery, []*Cheque, uint64, error) {
	var (
		total   uint64
		cheques []*Cheque
		entries []*merkletree.Entry
	)
	for index, p := range payees {
		entries = append(entries, &merkletree.Entry{
			Value:  p.Bytes(),
			Weight: weight[index],
		})
		total += weight[index]
	}
	tree, dropped := merkletree.NewMerkleTree(entries)
	if tree == nil {
		return nil, nil, 0, errors.New("empty tree")
	}
	var removed []int
	for index, payee := range payees {
		if _, ok := dropped[string(payee.Bytes())]; ok {
			removed = append(removed, index)
		}
	}
	for i := 0; i < len(removed); i++ {
		payees = append(payees[:removed[i]-i], payees[removed[i]-i+1:]...)
	}
	// New random lottery salt to ensure the id is unique.
	salt := rand.Uint64()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, salt)

	current := env.backend.Blockchain().CurrentHeader().Number.Uint64()
	lotteryId := crypto.Keccak256Hash(append(tree.Hash().Bytes(), buf...))
	lottery := &Lottery{
		Id:           lotteryId,
		RevealNumber: current + revealShift,
		Amount:       total,
		Receivers:    payees,
	}
	for _, entry := range entries {
		if _, ok := dropped[string(entry.Value)]; ok {
			continue
		}
		witness, _ := tree.Prove(entry)
		c, _ := newCheque(witness, env.contractAddr, salt, entry.Salt())
		cheques = append(cheques, c)
	}
	return lottery, cheques, salt, nil
}
