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
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestAddCheque(t *testing.T) {
	env := newTestEnv(t)
	defer env.close()

	var exit = make(chan struct{})
	defer close(exit)

	// Start the automatic blockchain.
	go func() {
		ticker := time.NewTicker(time.Millisecond * 10)
		for {
			select {
			case <-ticker.C:
				env.backend.Commit()
			case <-exit:
				return
			}
		}
	}()
	drawer, err := NewChequeDrawer(env.drawerAddr, env.contractAddr, bind.NewKeyedTransactor(env.drawerKey), nil, env.backend.Blockchain(), env.backend, env.backend, env.drawerDb)
	if err != nil {
		t.Fatalf("Faield to create drawer, err: %v", err)
	}
	defer drawer.Close()
	drawer.keySigner = func(data []byte) ([]byte, error) {
		sig, _ := crypto.Sign(data, env.drawerKey)
		return sig, nil
	}
	drawee, err := NewChequeDrawee(bind.NewKeyedTransactor(env.draweeKey), env.draweeAddr, drawer.ContractAddr(), env.backend.Blockchain(), env.backend, env.backend, env.draweeDb)
	if err != nil {
		t.Fatalf("Faield to create drawee, err: %v", err)
	}
	defer drawee.Close()
	// newLottery creates a valid lottery on-chain and generates
	// cheques with different testing requirements.
	newLottery := func(include bool, addition bool, reveal uint64) common.Hash {
		var payees []common.Address
		var amounts []uint64
		if include {
			payees = append(payees, env.draweeAddr)
			amounts = append(amounts, 128)
		}
		// Padding random payees
		if addition {
			for i := 0; i < 7; i++ {
				key, _ := crypto.GenerateKey()
				payees = append(payees, crypto.PubkeyToAddress(key.PublicKey))
				amounts = append(amounts, 128)
			}
		}
		id, _ := drawer.createLottery(context.Background(), payees, amounts, reveal, nil, nil)
		return id
	}
	// newRawCheque generates raw cheque to drawee with different testing
	// requirements.
	newRawCheque := func(key *ecdsa.PrivateKey) *Cheque {
		cheque, _ := newCheque([]common.Hash{crypto.Keccak256Hash(env.draweeAddr.Bytes())}, drawer.ContractAddr(), 10086, 0)
		cheque.signWithKey(func(digestHash []byte) ([]byte, error) {
			return crypto.Sign(digestHash, key)
		})
		return cheque
	}
	var lotteryID common.Hash
	var cases = []struct {
		genCheque    func() *Cheque
		expectErr    bool
		expectAmount uint64
	}{
		// Drawee should reject cheque if there is no corresponding deposit
		{func() *Cheque { return newRawCheque(env.drawerKey) }, true, 0},
		// Drawee should reject cheque if it's not the target payer
		{
			func() *Cheque {
				id := newLottery(false, true, 10086)
				cheques, _ := drawer.cdb.listCheques(env.drawerAddr, func(addr common.Address, lid common.Hash, cheque *Cheque) bool { return lid == id })
				return cheques[0]
			}, true, 0,
		},
		// Drawee should reject cheque if it's not signed by sender
		{
			func() *Cheque {
				cheque := drawer.cdb.readCheque(env.draweeAddr, env.drawerAddr, newLottery(true, true, 10086), true)
				cheque.Sig[60] = 100 // Modify the signature, the recovered signer should be changed.
				return cheque
			}, true, 0,
		},
		// Drawee should reject cheque if the net amount is zero
		{
			func() *Cheque {
				cheque := drawer.cdb.readCheque(env.draweeAddr, env.drawerAddr, newLottery(true, true, 10086), true)
				return cheque
			}, true, 0,
		},
		// Drawee should accept normal cheque
		{
			func() *Cheque {
				lotteryID = newLottery(true, true, 10086)
				cheque, _ := drawer.issueCheque(env.draweeAddr, lotteryID, 64, true)
				return cheque
			}, false, 64,
		},
		// Drawee should accept normal cheque with cumulative confirm amount
		{
			func() *Cheque {
				cheque, _ := drawer.issueCheque(env.draweeAddr, lotteryID, 64, true)
				return cheque
			}, false, 64,
		},
		// Drawee should accept normal cheque with the whole amount
		{
			func() *Cheque {
				cheque, _ := drawer.issueCheque(env.draweeAddr, newLottery(true, false, 10086), 128, true)
				return cheque
			}, false, 128,
		},
		// Drawee should reject cheque is the associated lottery is revealed(exceeds the safety threshold)
		{
			func() *Cheque {
				opt := bind.NewKeyedTransactor(env.drawerKey)
				opt.Value = big.NewInt(128)

				// New random lottery salt to ensure the id is unique.
				salt := drawer.rand.Uint64()
				buf := make([]byte, 8)
				binary.BigEndian.PutUint64(buf, salt)

				root := crypto.Keccak256Hash(env.draweeAddr.Bytes())
				id := crypto.Keccak256Hash(append(root.Bytes(), buf...))
				current := env.backend.Blockchain().CurrentHeader().Number.Uint64()

				tx, _ := drawer.book.contract.NewLottery(opt, id, current+30, salt)
				bind.WaitMined(context.Background(), drawer.dBackend, tx)

				cheque, _ := newCheque([]common.Hash{root}, drawer.ContractAddr(), salt, 0)
				cheque.RevealRange = []byte{0xff, 0xff, 0xff, 0xff}
				cheque.signWithKey(drawer.keySigner)
				cheque.deriveFields() // Recompute uint64 format reveal range

				for {
					if env.backend.Blockchain().CurrentBlock().NumberU64() >= current+30+lotterySafetyThreshold {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
				return cheque
			}, true, 0,
		},
		// Drawee should reject cheque if it doesn't belong to this channel
		{
			func() *Cheque {
				root := crypto.Keccak256Hash(env.draweeAddr.Bytes())
				cheque, _ := newCheque([]common.Hash{root}, common.HexToAddress("deadbeef"), 10086, 0)
				cheque.signWithKey(drawer.keySigner)
				return cheque
			}, true, 0,
		},
	}
	for _, c := range cases {
		amount, err := drawee.AddCheque(env.drawerAddr, c.genCheque())
		if c.expectErr {
			if err == nil {
				t.Fatalf("Expect error, not get nil")
			}
			continue
		} else if err != nil {
			t.Fatalf("Unexpect error: %v", err)
		}
		if c.expectAmount != amount {
			t.Fatalf("Net payment amount mismatch, want: %d, got: %d", c.expectAmount, amount)
		}
	}
}

func TestClaimLottery(t *testing.T) {
	env := newTestEnv(t)
	defer env.close()

	// Start the automatic blockchain.
	var exit = make(chan struct{})
	defer close(exit)
	go func() {
		ticker := time.NewTicker(time.Millisecond * 100)
		for {
			select {
			case <-ticker.C:
				env.backend.Commit()
			case <-exit:
				return
			}
		}
	}()
	drawer, err := NewChequeDrawer(env.drawerAddr, env.contractAddr, bind.NewKeyedTransactor(env.drawerKey), nil, env.backend.Blockchain(), env.backend, env.backend, env.drawerDb)
	if err != nil {
		t.Fatalf("Faield to create drawer, err: %v", err)
	}
	defer drawer.Close()
	drawer.keySigner = func(data []byte) ([]byte, error) {
		sig, _ := crypto.Sign(data, env.drawerKey)
		return sig, nil
	}
	drawee, err := NewChequeDrawee(bind.NewKeyedTransactor(env.draweeKey), env.draweeAddr, drawer.ContractAddr(), env.backend.Blockchain(), env.backend, env.backend, env.draweeDb)
	if err != nil {
		t.Fatalf("Faield to create drawee, err: %v", err)
	}
	defer drawee.Close()

	current := env.backend.Blockchain().CurrentHeader().Number.Uint64()
	id, err := drawer.createLottery(context.Background(), []common.Address{env.draweeAddr}, []uint64{128}, current+30, nil, nil)
	if err != nil {
		t.Fatalf("Faield to create lottery, err: %v", err)
	}
	cheque, err := drawer.issueCheque(env.draweeAddr, id, 128, true)
	if err != nil {
		t.Fatalf("Faield to create cheque, err: %v", err)
	}
	done := make(chan struct{}, 1)
	drawee.onClaimedHook = func(id common.Hash) {
		if id == cheque.LotteryId {
			done <- struct{}{}
		}
	}
	drawee.AddCheque(env.drawerAddr, cheque)
	select {
	case <-done:
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatalf("timeout")
	}
}
