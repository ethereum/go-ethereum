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

package contract

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/merkletree"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

func init() {
	rand.Seed(int64(time.Now().Second()))
}

type Account struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

type merkleEntry struct {
	chance  uint64
	account Account
}

type testLotteryBook struct {
	sim      *backends.SimulatedBackend
	contract *LotteryBook
	address  common.Address
	deployer Account
	sender   Account
	receiver Account
}

func newAccount() Account {
	key, _ := crypto.GenerateKey()
	return Account{addr: crypto.PubkeyToAddress(key.PublicKey), key: key}
}

func newTestLotteryBook(t *testing.T) *testLotteryBook {
	deployer, sender, receiver := newAccount(), newAccount(), newAccount()

	sim := backends.NewSimulatedBackend(core.GenesisAlloc{
		deployer.addr: {Balance: big.NewInt(2e18)},
		sender.addr:   {Balance: big.NewInt(2e18)},
		receiver.addr: {Balance: big.NewInt(1000000000)},
	}, 10000000)
	addr, _, c, err := DeployLotteryBook(bind.NewKeyedTransactor(deployer.key), sim)
	if err != nil {
		t.Error("Failed to deploy registrar contract", err)
	}
	sim.Commit()

	return &testLotteryBook{
		sim:      sim,
		contract: c,
		address:  addr,
		deployer: deployer,
		sender:   sender,
		receiver: receiver,
	}
}

func (tester *testLotteryBook) newMerkleTree(entries []merkleEntry) (*merkletree.MerkleTree, []*merkletree.Entry, error) {
	var merkleEntries []*merkletree.Entry
	for _, entry := range entries {
		merkleEntries = append(merkleEntries, &merkletree.Entry{
			Value:  entry.account.addr.Bytes(),
			Weight: entry.chance,
		})
	}
	tree, dropped := merkletree.NewMerkleTree(merkleEntries)
	if tree == nil {
		return nil, nil, errors.New("invalid entry")
	}
	var removed []int
	for index, entry := range merkleEntries {
		if _, ok := dropped[string(entry.Value)]; ok {
			removed = append(removed, index)
		}
	}
	for i := 0; i < len(removed); i++ {
		merkleEntries = append(merkleEntries[:removed[i]-i], merkleEntries[removed[i]-i+1:]...)
	}
	return tree, merkleEntries, nil
}

func (tester *testLotteryBook) issueCheque(id common.Hash, revealRange [4]byte) []byte {
	var appContent []byte
	appContent = append(appContent, id.Bytes()...)
	appContent = append(appContent, revealRange[:]...)
	data := append([]byte{0x19, 0x00}, append(tester.address.Bytes(), appContent...)...)
	sig, _ := crypto.Sign(crypto.Keccak256(data), tester.sender.key)
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return sig
}

// calcuValidRange returns a wining ranges based on the probabilty range and
// provided percentage.
func (tester *testLotteryBook) calcuValidRange(pos uint64, level uint64, percentage float64) (uint64, uint64) {
	interval := uint64((math.MaxUint32 + 1) >> level)
	start := interval * pos
	end := interval*pos + uint64(float64(interval)*percentage) - 1
	return start, end
}

func (tester *testLotteryBook) commitEmptyBlocks(number int) {
	for i := 0; i < number; i++ {
		tester.sim.Commit()
	}
}

func (tester *testLotteryBook) commitEmptyUntil(end uint64) {
	for {
		if tester.sim.Blockchain().CurrentHeader().Number.Uint64() == end {
			return
		}
		tester.sim.Commit()
	}
}

func (tester *testLotteryBook) teardown() {
	tester.sim.Close()
}

func TestNewLottery(t *testing.T) {
	tester := newTestLotteryBook(t)
	defer tester.teardown()

	submitAndCheck := func(owner bool, id [32]byte, amount uint64, revealNumber uint64, salt uint64, expectErr bool) {
		var opt *bind.TransactOpts
		if owner {
			opt = bind.NewKeyedTransactor(tester.sender.key)
			opt.Value = big.NewInt(int64(amount))
		} else {
			opt = bind.NewKeyedTransactor(tester.receiver.key)
			opt.Value = big.NewInt(int64(amount))
		}
		_, err := tester.contract.NewLottery(opt, id, revealNumber, salt)
		if expectErr {
			if err == nil {
				t.Fatalf("Expect to catch error, but nil")
			}
			return
		}
		if err != nil {
			t.Fatalf("Failed to submit new lottery: %v", err)
		}
		tester.sim.Commit()

		lottery, err := tester.contract.Lotteries(nil, id)
		if err != nil {
			t.Fatalf("Failed to retrieve lottery: %v", err)
		}
		if lottery.Amount != amount {
			t.Fatalf("Amount mismatch, want: %d, got: %d", amount, lottery.Amount)
		}
		if lottery.RevealNumber != revealNumber {
			t.Fatalf("Reveal number mismatch, want: %d, got: %d", revealNumber, lottery.RevealNumber)
		}
		if lottery.Salt != salt {
			t.Fatalf("Salt mismatch, want: %d, got: %d", salt, lottery.Salt)
		}
	}
	// Valid lotteries
	submitAndCheck(true, common.HexToHash("deadbeef0"), 100, 100, 100, false)  // Normal submission should succeed
	submitAndCheck(true, common.HexToHash("deadbeef1"), 1e18, 100, 100, false) // 1Ether lottery should be accepted
	submitAndCheck(true, common.HexToHash("deadbeef2"), 100, 100, 100, false)  // Different lotteries are allowed to submit at same reveal number

	// Invalid lottery amount
	submitAndCheck(true, common.HexToHash("deadbeef3"), 0, 100, 100, true)      // Empty lottery should be rejected
	submitAndCheck(true, common.HexToHash("deadbeef3"), 1e18+1, 100, 100, true) // Large lottery should be rejected
	submitAndCheck(true, common.HexToHash("deadbeef3"), 2e18, 100, 100, true)   // Large lottery should be rejected

	// Invalid lottery id
	submitAndCheck(true, common.HexToHash("deadbeef2"), 100, 100, 200, true) // Duplicated lottery should be rejected

	// Invalid lottery reveal number
	submitAndCheck(true, common.HexToHash("deadbeef3"), 100, 1, 200, true) // Stale lottery should be rejected
	now := tester.sim.Blockchain().CurrentHeader().Number.Uint64()
	submitAndCheck(true, common.HexToHash("deadbeef3"), 100, now, 200, true) // Useless lottery should be rejected

	// Different lottery submitter
	submitAndCheck(false, common.HexToHash("deadbeef3"), 100, 100, 100, false) // Other user is also allowed to submit
}

func TestClaimLottery(t *testing.T) {
	tester := newTestLotteryBook(t)
	defer tester.teardown()

	claimAndCheck := func(entries []merkleEntry, percentages []float64) {
		tree, merkleEntries, err := tester.newMerkleTree(entries)
		if err != nil {
			t.Fatalf("Failed to construct merkle tree: %v", err)
		}
		opt := bind.NewKeyedTransactor(tester.sender.key)
		opt.Value = big.NewInt(100000)

		// Compute id based on a random salt and merkle root
		salt := rand.Uint64()
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, salt)
		id := crypto.Keccak256Hash(append(tree.Hash().Bytes(), buf...))

		// Submit the lottery which includes all receivers
		revealNumber := tester.sim.Blockchain().CurrentHeader().Number.Uint64() + 5
		_, err = tester.contract.NewLottery(opt, id, revealNumber, salt)
		if err != nil {
			t.Fatalf("Failed to submit new lottery: %v", err)
		}
		tester.sim.Commit()
		tester.commitEmptyBlocks(5)

		// Everybody check do I win?
		block := tester.sim.Blockchain().GetBlockByNumber(revealNumber)
		revealhash := block.Hash()

		// Use the highest four bytes in big-endian order to construct reveal number.
		var trimmed [4]byte
		copy(trimmed[:], revealhash.Bytes()[common.HashLength-4:])
		value := uint64(binary.BigEndian.Uint32(trimmed[:]))

		var claimed bool
	loop:
		for _, e := range merkleEntries {
			proof, err := tree.Prove(e)
			if err != nil {
				t.Fatalf("Failed to generate merkle proof: %v", err)
			}
			pos, err := merkletree.VerifyProof(tree.Hash(), proof)
			if err != nil {
				t.Fatalf("Invalid merkle proof: %v", err)
			}
			for _, p := range percentages {
				start, end := tester.calcuValidRange(pos, e.Level(), p)
				if start > value || end < value {
					continue
				}
				for _, a := range entries {
					if bytes.Equal(e.Value, a.account.addr.Bytes()) {
						original, err := tester.sim.BalanceAt(context.Background(), a.account.addr, nil)
						if err != nil {
							t.Fatalf("Failed to retrieve balance: %v", err)
						}
						opt := bind.NewKeyedTransactor(a.account.key)
						opt.GasPrice = big.NewInt(0)

						var proofslice [][32]byte
						for i, h := range proof {
							if i == 0 {
								continue
							}
							var p [32]byte
							copy(p[:], h.Bytes())
							proofslice = append(proofslice, p)
						}
						var signedRange [4]byte
						binary.BigEndian.PutUint32(signedRange[:], uint32(end))
						sig := tester.issueCheque(id, signedRange)
						_, err = tester.contract.Claim(opt, id, signedRange, sig[64], common.BytesToHash(sig[:32]), common.BytesToHash(sig[32:64]), e.Salt(), proofslice)
						if err != nil {
							t.Fatalf("Failed to claim lottery: %v", err)
						}
						claimed = true
						tester.sim.Commit()

						latest, err := tester.sim.BalanceAt(context.Background(), a.account.addr, nil)
						if err != nil {
							t.Fatalf("Failed to retrieve balance: %v", err)
						}
						if latest.Uint64()-original.Uint64() != 100000 {
							t.Fatalf("Failed to claim whole deposit")
						}
						break loop
					}
				}
			}
		}
		if !claimed {
			t.Fatal("Lottery is not claimed")
		}
	}
	// Check must have one receiver can claim the lottery if signed range is full.
	for i := 0; i < 100; i++ {
		var (
			length  = rand.Intn(30) + 1
			entries []merkleEntry
		)
		for i := 0; i < length; i++ {
			entries = append(entries, merkleEntry{
				chance:  uint64(rand.Intn(30) + 1),
				account: newAccount(),
			})
		}
		claimAndCheck(entries, []float64{1})
	}
	// Check the specified receiver must can claim the lottery
	// if there is only one receiver and the posibility will
	// reach 100% finally.
	claimAndCheck([]merkleEntry{{1, tester.receiver}}, []float64{0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875, 1})
}

func TestResetLottery(t *testing.T) {
	tester := newTestLotteryBook(t)
	defer tester.teardown()

	opt := bind.NewKeyedTransactor(tester.sender.key)
	opt.Value = big.NewInt(100)

	current := tester.sim.Blockchain().CurrentHeader().Number.Uint64()
	_, err := tester.contract.NewLottery(opt, common.HexToHash("deadbeef"), current+5, 10) // The reveal point is 4 blocks after
	if err != nil {
		t.Fatalf("Failed to submit new lottery: %v", err)
	}
	tester.sim.Commit()

	var cases = []struct {
		shift     uint64
		amount    uint64
		expectErr bool
	}{
		{current + 5 + 255, 100, true},            // Reveal+256, last chance to claim
		{current + 5 + 256, 1e18 - 100 + 1, true}, // Reveal+257, first time to reown, invalid amount
		{current + 5 + 256, 1e18 - 100, false},    // Reveal+257, first time to reown
	}
	for index, c := range cases {
		tester.commitEmptyUntil(c.shift)
		opt := bind.NewKeyedTransactor(tester.sender.key)
		opt.Value = big.NewInt(int64(c.amount))
		_, err := tester.contract.ResetLottery(opt, common.HexToHash("deadbeef"), common.HexToHash("deadbeef2"), 10086, 20)
		if c.expectErr && err == nil {
			t.Fatalf("Case %d, expect error, but nil", index)
		}
		if !c.expectErr && err != nil {
			t.Fatalf("Case %d, expect no error, but occurs: %v", index, err)
		}
		if !c.expectErr {
			tester.sim.Commit()
			lottery, err := tester.contract.Lotteries(nil, common.HexToHash("deadbeef2"))
			if err != nil {
				t.Fatalf("Failed to retrieve lottery: %v", err)
			}
			if lottery.RevealNumber != 10086 {
				t.Fatalf("Reveal number mismatch, want: %d, got: %d", 10086, lottery.RevealNumber)
			}
			if lottery.Salt != 20 {
				t.Fatalf("Salt mismatch, want: %d, got: %d", 20, lottery.Salt)
			}
			if lottery.Amount != c.amount+100 {
				t.Fatalf("Amount mismatch, want: %d, got: %d", c.amount+100, lottery.Amount)
			}
			stale, err := tester.contract.Lotteries(nil, common.HexToHash("deadbeef"))
			if err != nil {
				t.Fatalf("Failed to retrieve lottery: %v", err)
			}
			if stale.Amount != 0 {
				t.Fatalf("Failed to delete stale lottery")
			}
		}
	}
	_, err = tester.contract.ResetLottery(opt, common.HexToHash("deadbeef3"), common.HexToHash("deadbeef4"), 10086, 20)
	if err == nil {
		t.Fatalf("Reset non-existent lottery should be rejected")
	}
}

func TestResetAndReset(t *testing.T) {
	tester := newTestLotteryBook(t)
	defer tester.teardown()

	opt := bind.NewKeyedTransactor(tester.sender.key)
	opt.Value = big.NewInt(100)

	current := tester.sim.Blockchain().CurrentHeader().Number.Uint64()
	_, err := tester.contract.NewLottery(opt, common.HexToHash("deadbeef"), current+5, 10) // The reveal point is 4 blocks after
	if err != nil {
		t.Fatalf("Failed to submit new lottery: %v", err)
	}
	tester.sim.Commit()
	tester.commitEmptyBlocks(10 + 256)

	current = tester.sim.Blockchain().CurrentHeader().Number.Uint64()
	_, err = tester.contract.ResetLottery(opt, common.HexToHash("deadbeef"), common.HexToHash("deadbeef1"), current+5, 20)
	if err != nil {
		t.Fatalf("Failed to reset lottery: %v", err)
	}
	tester.sim.Commit()
	tester.commitEmptyBlocks(10 + 256)

	_, err = tester.contract.ResetLottery(opt, common.HexToHash("deadbeef1"), common.HexToHash("deadbeef2"), 10086, 20)
	if err != nil {
		t.Fatalf("Failed to double reset lottery: %v", err)
	}
}

func TestDestoryLottery(t *testing.T) {
	tester := newTestLotteryBook(t)
	defer tester.teardown()

	opt := bind.NewKeyedTransactor(tester.sender.key)
	opt.Value = big.NewInt(100)

	current := tester.sim.Blockchain().CurrentHeader().Number.Uint64()
	_, err := tester.contract.NewLottery(opt, common.HexToHash("deadbeef"), current+5, 10) // The reveal point is 4 blocks after
	if err != nil {
		t.Fatalf("Failed to submit new lottery: %v", err)
	}
	tester.sim.Commit()

	var cases = []struct {
		shift     uint64
		expectErr bool
	}{
		{current + 5 + 255, true},  // Reveal+256, last chance to claim
		{current + 5 + 256, false}, // Reveal+257, first time to reown
	}
	for index, c := range cases {
		tester.commitEmptyUntil(c.shift)

		opt := bind.NewKeyedTransactor(tester.sender.key)
		opt.GasPrice = big.NewInt(0)
		balance, err := tester.sim.BalanceAt(context.Background(), tester.sender.addr, nil)
		if err != nil {
			t.Fatalf("Failed to retrieve balance: %v", err)
		}
		_, err = tester.contract.DestroyLottery(opt, common.HexToHash("deadbeef"))
		if c.expectErr && err == nil {
			t.Fatalf("Case %d, expect error, but nil", index)
		}
		if !c.expectErr && err != nil {
			t.Fatalf("Case %d, expect no error, but occurs", index)
		}
		if !c.expectErr {
			tester.sim.Commit()
			lottery, err := tester.contract.Lotteries(nil, common.HexToHash("deadbeef"))
			if err != nil {
				t.Fatalf("Failed to retrieve lottery: %v", err)
			}
			if lottery.Amount != 0 {
				t.Fatal("Lottery should be destoryed")
			}
			balance2, err := tester.sim.BalanceAt(context.Background(), tester.sender.addr, nil)
			if err != nil {
				t.Fatalf("Failed to retrieve balance: %v", err)
			}
			if new(big.Int).Sub(balance2, balance).Uint64() != 100 {
				t.Fatalf("Deposit should be refund, new: %d, old: %d", balance2, balance)
			}
		}
	}
	_, err = tester.contract.DestroyLottery(opt, common.HexToHash("deadbeef1"))
	if err == nil {
		t.Fatalf("Destroy non-existent lottery should be rejected")
	}
}
