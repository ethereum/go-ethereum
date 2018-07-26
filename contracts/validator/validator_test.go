// Copyright (c) 2018 Tomochain
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package validator

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/contracts"
	"github.com/ethereum/go-ethereum/contracts/validator/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"math/rand"
	"time"
)

var (
	key, _     = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr       = crypto.PubkeyToAddress(key.PublicKey)
	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc3Key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc4Key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee04aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)
	acc3Addr   = crypto.PubkeyToAddress(acc3Key.PublicKey)
	acc4Addr   = crypto.PubkeyToAddress(acc4Key.PublicKey)
)

func TestValidator(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	transactOpts := bind.NewKeyedTransactor(key)

	_, validator, err := DeployValidator(transactOpts, contractBackend)
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}
	contractBackend.Commit()

	candidates, err := validator.GetCandidates()
	if err != nil {
		t.Fatalf("can't get candidates: %v", err)
	}
	for _, it := range candidates {
		cap, _ := validator.GetCandidateCap(it)
		t.Log("candidate", it.String(), "cap", cap)
		owner, _ := validator.GetCandidateOwner(it)
		t.Log("candidate", it.String(), "validator owner", owner.String())
	}
	contractBackend.Commit()
}

func TestRewardBalance(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{
		acc1Addr: {Balance: new(big.Int).SetUint64(10000000)},
		acc2Addr: {Balance: new(big.Int).SetUint64(10000000)},
		acc4Addr: {Balance: new(big.Int).SetUint64(10000000)},
	})
	acc1Opts := bind.NewKeyedTransactor(acc1Key)
	acc2Opts := bind.NewKeyedTransactor(acc2Key)
	accounts := []*bind.TransactOpts{acc1Opts, acc2Opts}
	transactOpts := bind.NewKeyedTransactor(acc1Key)

	validatorAddr, _, baseValidator, err := contract.DeployTomoValidator(transactOpts, contractBackend, big.NewInt(50000), big.NewInt(99), big.NewInt(100))
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}
	contractBackend.Commit()

	// Propose master node acc3Addr.
	opts := bind.NewKeyedTransactor(acc4Key)
	opts.Value = new(big.Int).SetUint64(50000)
	acc4Validator, _ := NewValidator(opts, validatorAddr, contractBackend)
	acc4Validator.Propose(acc3Addr, "http://")
	contractBackend.Commit()

	totalVote := 0
	type logCap struct {
		Addr    string
		Balance int
	}
	logCaps := make(map[int]*logCap)
	for i := 0; i <= 10; i++ {
		rand.Seed(time.Now().UTC().UnixNano())
		randIndex := rand.Intn(len(accounts))
		randCap := rand.Intn(10) * 1000
		if randCap <= 0 {
			randCap = 1000
		}
		totalVote += randCap
		accounts[randIndex].Value = new(big.Int).SetInt64(int64(randCap))
		validator, err := NewValidator(accounts[randIndex], validatorAddr, contractBackend)
		if err != nil {
			t.Fatalf("can't get current validator: %v", err)
		}
		validator.Vote(acc3Addr)
		contractBackend.Commit()
		logCaps[i] = &logCap{accounts[randIndex].From.String(), randCap}
	}

	totalReward := new(big.Int).SetInt64(15 * 1000)
	rewards, err := contracts.GetRewardBalancesRate(acc3Addr, totalReward, baseValidator)
	if err != nil {
		t.Error("Fail to get reward balances rate.", err)
	}

	afterReward := new(big.Int)
	for _, value := range rewards {
		afterReward = new(big.Int).Add(afterReward, value)
	}

	if totalReward.Int64()+1 < afterReward.Int64() || totalReward.Int64()-1 > afterReward.Int64() {
		callOpts := new(bind.CallOpts)
		voters, err := baseValidator.GetVoters(callOpts, acc3Addr)
		if err != nil {
			t.Fatal("Can not get voters in validator contract.", err)
		}
		for addr, capacity := range logCaps {
			t.Errorf("from %v - %v", addr, capacity)
		}
		for _, voter := range voters {
			voteCap, _ := baseValidator.GetVoterCap(callOpts, acc3Addr, voter)
			t.Errorf("vote %v - %v", voter.String(), voteCap)
		}
		for addr, value := range rewards {
			t.Errorf("reaward %v - %v", addr.String(), value)
		}

		t.Errorf("reward total %v - %v", totalReward, afterReward)
	}

}
