package contracts

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"math/rand"
	"testing"
)

func TestSendTxSign(t *testing.T) {
	acc1Key, _ := crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ := crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc3Key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc4Key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee04aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr := crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr := crypto.PubkeyToAddress(acc2Key.PublicKey)
	acc3Addr := crypto.PubkeyToAddress(acc3Key.PublicKey)
	acc4Addr := crypto.PubkeyToAddress(acc4Key.PublicKey)
	accounts := []common.Address{acc2Addr, acc3Addr, acc4Addr}
	keys := []*ecdsa.PrivateKey{acc2Key, acc3Key, acc4Key}

	signer := types.HomesteadSigner{}
	genesis := core.GenesisAlloc{acc1Addr: {Balance: big.NewInt(1000000000)}}
	backend := backends.NewSimulatedBackend(genesis)
	backend.Commit()
	ctx := context.Background()

	transactOpts := bind.NewKeyedTransactor(acc1Key)
	blockSignerAddr, blockSigner, err := blocksigner.DeployBlockSigner(transactOpts, backend)
	if err != nil {
		t.Fatalf("Can't get block signer: %v", err)
	}
	backend.Commit()

	nonces := make(map[*ecdsa.PrivateKey]int)
	oldBlock := make([]common.Address, 100)

	signTx := func(ctx context.Context, backend *backends.SimulatedBackend, signer types.HomesteadSigner, nonces map[*ecdsa.PrivateKey]int, accKey *ecdsa.PrivateKey, i uint64) {
		tx, _ := types.SignTx(CreateTxSign(new(big.Int).SetUint64(i), uint64(nonces[accKey]), blockSignerAddr), signer, accKey)
		backend.SendTransaction(ctx, tx)
		backend.Commit()
		nonces[accKey]++
	}

	// Tx sign for signer.
	signCount := uint64(0)
	for i := uint64(0); i < 100; i++ {
		randIndex := rand.Intn(len(keys))
		accKey := keys[randIndex]
		signTx(ctx, backend, signer, nonces, accKey, i)
		oldBlock[i] = accounts[randIndex]
		signCount++

		// Tx sign for validators.
		for _, key := range keys {
			if key != accKey {
				signTx(ctx, backend, signer, nonces, key, i)
				signCount++
			}
		}
	}

	for i := uint64(0); i < 100; i++ {
		signers, err := blockSigner.GetSigners(new(big.Int).SetUint64(i))
		if err != nil {
			t.Fatalf("Can't get signers: %v", err)
		}

		if signers[0].String() != oldBlock[i].String() {
			t.Errorf("Tx sign for block signer not match %v - %v", signers[0].String(), oldBlock[i].String())
		}

		if len(signers) != len(keys) {
			t.Error("Tx sign for block validators not match")
		}
	}

	// Unit test for reward checkpoint.
	rCheckpoint := uint64(5)
	chainReward := new(big.Int).SetUint64(15 * params.Ether)
	total := new(uint64)
	for i := uint64(0); i < 100; i++ {
		if i > 0 && i%rCheckpoint == 0 && i-rCheckpoint > 0 {
			_, err := GetRewardForCheckpoint(blockSignerAddr, i, rCheckpoint, backend, total)
			if err != nil {
				t.Errorf("Fail to get signers for reward checkpoint: %v", err)
			}
		}
	}

	signers := make(map[common.Address]*rewardLog)
	totalSigner := uint64(17)
	signers[common.HexToAddress("0x12f588d7d03bb269b382b842fc15d874e8c055a7")] = &rewardLog{5, new(big.Int).SetUint64(0)}
	signers[common.HexToAddress("0x1f9e122c0921a4504fc116d967baf7a7bf2604ef")] = &rewardLog{6, new(big.Int).SetUint64(0)}
	signers[common.HexToAddress("0xea489e4e673c25ff0614617ebe88efd853efe00c")] = &rewardLog{6, new(big.Int).SetUint64(0)}
	rewardSigners, err := CalculateReward(chainReward, signers, totalSigner)
	if err != nil {
		t.Errorf("Fail to calculate reward for signers: %v", err)
	}
	//t.Error("Reward", rewardSigners)
	rewards := new(big.Int)
	for _, reward := range rewardSigners {
		rewards.Add(rewards, reward)
	}
	if rewards.Cmp(new(big.Int).SetUint64(14999999999999999996)) != 0 {
		t.Errorf("Total reward not same reward checkpoint: %v - %v", chainReward, rewards)
	}
}
