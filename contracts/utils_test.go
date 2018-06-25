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
	acc1Addr := crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr := crypto.PubkeyToAddress(acc2Key.PublicKey)
	acc3Addr := crypto.PubkeyToAddress(acc3Key.PublicKey)
	accounts := []common.Address{acc2Addr, acc3Addr}
	keys := []*ecdsa.PrivateKey{acc2Key, acc3Key}

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
	for i := uint64(0); i < 100; i++ {
		randIndex := rand.Intn(len(keys))
		accKey := keys[randIndex]
		signTx(ctx, backend, signer, nonces, accKey, i)
		oldBlock[i] = accounts[randIndex]

		// Tx sign for validators.
		for _, key := range keys {
			if key != accKey {
				signTx(ctx, backend, signer, nonces, key, i)
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
	rCheckpoint := uint64(10)
	chainReward := new(big.Int).SetUint64(15 * params.Ether)
	for i := uint64(0); i < 100; i++ {
		if i > 0 && i%rCheckpoint == 0 && i-rCheckpoint > 0 {
			signers, err := GetRewardForCheckpoint(chainReward, blockSignerAddr, i, rCheckpoint, backend)
			if err != nil {
				t.Errorf("Fail to get signers for reward checkpoint: %v", err)
			}
			rewards := new(big.Int)
			for _, reward := range signers {
				rewards.Add(rewards, reward)
			}
			if rewards.Cmp(chainReward) != 0 {
				t.Errorf("Total reward not same reward checkpoint: %v - %v", chainReward, rewards)
			}
		}
	}
}
