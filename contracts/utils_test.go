// Copyright (c) 2018 XDPoSChain
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

package contracts

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/contracts/blocksigner"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/params"
)

var (
	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc3Key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc4Key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee04aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)
	acc3Addr   = crypto.PubkeyToAddress(acc3Key.PublicKey)
	acc4Addr   = crypto.PubkeyToAddress(acc4Key.PublicKey)
)

func getCommonBackend() *backends.SimulatedBackend {
	genesis := types.GenesisAlloc{acc1Addr: {Balance: big.NewInt(1000000000000)}}
	backend := backends.NewXDCSimulatedBackend(genesis, 10000000, params.TestXDPoSMockChainConfig)
	backend.Commit()

	return backend
}

func TestSendTxSign(t *testing.T) {
	accounts := []common.Address{acc2Addr, acc3Addr, acc4Addr}
	keys := []*ecdsa.PrivateKey{acc2Key, acc3Key, acc4Key}
	backend := getCommonBackend()
	signer := types.HomesteadSigner{}
	ctx := context.Background()

	transactOpts := bind.NewKeyedTransactor(acc1Key)
	blockSignerAddr, blockSigner, err := blocksigner.DeployBlockSigner(transactOpts, backend, big.NewInt(99))
	if err != nil {
		t.Fatalf("Can't get block signer: %v", err)
	}
	backend.Commit()

	nonces := make(map[*ecdsa.PrivateKey]int)
	oldBlocks := make(map[common.Hash]common.Address)

	signTx := func(ctx context.Context, backend *backends.SimulatedBackend, signer types.HomesteadSigner, nonces map[*ecdsa.PrivateKey]int, accKey *ecdsa.PrivateKey, blockNumber *big.Int, blockHash common.Hash) *types.Transaction {
		tx, _ := types.SignTx(CreateTxSign(blockNumber, blockHash, uint64(nonces[accKey]), blockSignerAddr), signer, accKey)
		backend.SendTransaction(ctx, tx)
		backend.Commit()
		nonces[accKey]++

		return tx
	}

	// Tx sign for signer.
	signCount := int64(0)
	blockHashes := make([]common.Hash, 10)
	for i := int64(0); i < 10; i++ {
		blockHash := randomHash()
		blockHashes[i] = blockHash
		randIndex := rand.Intn(len(keys))
		accKey := keys[randIndex]
		signTx(ctx, backend, signer, nonces, accKey, new(big.Int).SetInt64(i), blockHash)
		oldBlocks[blockHash] = accounts[randIndex]
		signCount++

		// Tx sign for validators.
		for _, key := range keys {
			if key != accKey {
				signTx(ctx, backend, signer, nonces, key, new(big.Int).SetInt64(i), blockHash)
				signCount++
			}
		}
	}

	for _, blockHash := range blockHashes {
		signers, err := blockSigner.GetSigners(blockHash)
		if err != nil {
			t.Fatalf("Can't get signers: %v", err)
		}

		if signers[0] != oldBlocks[blockHash] {
			t.Errorf("Tx sign for block signer not match %v - %v", signers[0], oldBlocks[blockHash])
		}

		if len(signers) != len(keys) {
			t.Error("Tx sign for block validators not match")
		}
	}
}

// Generate random string.
func randomHash() common.Hash {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	var b common.Hash
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

// Unit test for get random position of masternodes.
func TestRandomMasterNode(t *testing.T) {
	oldSlice := NewSlice(0, 10, 1)
	newSlice := Shuffle(oldSlice)
	for _, newNumber := range newSlice {
		for i, oldNumber := range oldSlice {
			if oldNumber == newNumber {
				// Delete find element.
				oldSlice = append(oldSlice[:i], oldSlice[i+1:]...)
			}
		}
	}
	if len(oldSlice) != 0 {
		t.Errorf("Test generate random masternode fail %v - %v", oldSlice, newSlice)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	//byteInteger := common.LeftPadBytes([]byte(new(big.Int).SetInt64(4).String()), 32)
	randomByte := RandStringByte(32)
	encrypt := Encrypt(randomByte, new(big.Int).SetInt64(4).String())
	decrypt := Decrypt(randomByte, encrypt)
	t.Log("Encrypt", encrypt, "Test", string(randomByte), "Decrypt", decrypt, "trim", string(bytes.TrimLeft([]byte(decrypt), "\x00")))
}

func isArrayEqual(a [][]int64, b [][]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, vs := range a {
		for j, v := range vs {
			if v != b[i][j] {
				return false
			}
		}
	}
	return true
}

// Unit test for
func TestGenM2FromRandomize(t *testing.T) {
	a := make([]int64, 0, 11)
	for i := 0; i <= 10; i++ {
		a = append(a, int64(rand.Intn(9999)))
	}
	b, err := GenM2FromRandomize(a, common.MaxMasternodes)
	t.Log("randomize", b, "len", len(b))
	if err != nil {
		t.Error("Fail to test gen m2 for randomize.", err)
	}
	// Test Permutation Without Fixed-point.
	M1List := NewSlice(int64(0), common.MaxMasternodes, 1)
	for i, m1 := range M1List {
		if m1 == b[i] {
			t.Errorf("Error check Permutation Without Fixed-point %v - %v - %v", i, b[i], a)
		}
	}
}

// Unit test for validator m2.
func TestBuildValidatorFromM2(t *testing.T) {
	a := []int64{84, 58, 27, 96, 127, 60, 136, 20, 121, 31, 87, 85, 40, 120, 149, 109, 141, 145, 11, 110, 147, 35, 76, 46, 34, 108, 72, 103, 102, 12, 23, 47, 70, 86, 125, 112, 128, 13, 130, 98, 126, 62, 132, 111, 134, 6, 106, 67, 24, 91, 101, 50, 94, 43, 77, 73, 129, 71, 51, 10, 92, 29, 80, 95, 33, 100, 124, 75, 38, 133, 79, 83, 61, 36, 122, 99, 16, 28, 18, 116, 140, 97, 119, 82, 148, 48, 56, 32, 93, 107, 69, 68, 123, 81, 22, 137, 25, 115, 44, 8, 42, 131, 143, 17, 55, 89, 9, 15, 19, 59, 146, 54, 5, 30, 41, 144, 117, 1, 104, 49, 105, 45, 88, 78, 74, 135, 0, 21, 57, 3, 66, 52, 63, 138, 4, 114, 37, 118, 14, 2, 26, 7, 65, 139, 39, 64, 90, 142, 53, 113}
	b := BuildValidatorFromM2(a)
	c, _ := utils.ExtractValidatorsFromBytes(b)
	if !isArrayEqual([][]int64{a}, [][]int64{c}) {
		t.Errorf("Fail to get m2 result %v", b)
	}
}

// Unit test for decode validator string data.
func TestDecodeValidatorsHexData(t *testing.T) {
	a := "0x000000310000003000000032000000310000003000000032000000310000003000000032000000310000003000000031000000320000003000000031000000320000003000000031000000320000003000000030000000310000003200000030000000310000003200000030000000310000003200000030000000300000003100000032000000300000003100000032000000300000003100000032000000300000003200000030000000310000003200000030000000310000003200000030000000310000003000000030"
	b, err := DecodeValidatorsHexData(a)
	if err != nil {
		t.Error("Fail to decode validator from hex string", err)
	}
	c := []int64{1, 0, 2, 1, 0, 2, 1, 0, 2, 1, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 2, 0, 1, 2, 0, 1, 2, 0, 1, 0, 0}
	if !isArrayEqual([][]int64{b}, [][]int64{c}) {
		t.Errorf("Fail to get m2 result %v", b)
	}
	t.Log("b", b)
}

type createTxSignTestChain struct{}

func (createTxSignTestChain) Config() *params.ChainConfig { return params.TestChainConfig }

func (createTxSignTestChain) CurrentBlock() *types.Header {
	return &types.Header{Number: big.NewInt(0)}
}

func (createTxSignTestChain) StateAt(common.Hash) (*state.StateDB, error) {
	return state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
}

func (createTxSignTestChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

type nonceGuardSubPool struct {
	seededNonce0 bool
	added        []*types.Transaction
}

func (s *nonceGuardSubPool) Filter(tx *types.Transaction) bool { return true }

func (s *nonceGuardSubPool) Init(gasTip uint64, head *types.Header, reserver txpool.Reserver) error {
	return nil
}

func (s *nonceGuardSubPool) Close() error { return nil }

func (s *nonceGuardSubPool) Reset(oldHead, newHead *types.Header) {}

func (s *nonceGuardSubPool) SetGasTip(tip *big.Int) error { return nil }

func (s *nonceGuardSubPool) Has(hash common.Hash) bool { return false }

func (s *nonceGuardSubPool) Get(hash common.Hash) *types.Transaction { return nil }

func (s *nonceGuardSubPool) ValidateTxBasics(tx *types.Transaction) error { return nil }

func (s *nonceGuardSubPool) Add(txs []*types.Transaction, sync bool) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		if tx.Nonce() == 0 && s.seededNonce0 {
			errs[i] = txpool.ErrReplaceUnderpriced
			continue
		}
		s.added = append(s.added, tx)
		if tx.Nonce() == 0 {
			s.seededNonce0 = true
		}
	}
	return errs
}

func (s *nonceGuardSubPool) Pending(filter txpool.PendingFilter) map[common.Address][]*txpool.LazyTransaction {
	return map[common.Address][]*txpool.LazyTransaction{}
}

func (s *nonceGuardSubPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

func (s *nonceGuardSubPool) Nonce(addr common.Address) uint64 { return 1 }

func (s *nonceGuardSubPool) Stats() (int, int) { return 0, 0 }

func (s *nonceGuardSubPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return map[common.Address][]*types.Transaction{}, map[common.Address][]*types.Transaction{}
}

func (s *nonceGuardSubPool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return nil, nil
}

func (s *nonceGuardSubPool) Status(hash common.Hash) txpool.TxStatus { return txpool.TxStatusUnknown }

func (s *nonceGuardSubPool) SetSigner(f func(address common.Address) bool) {}

func (s *nonceGuardSubPool) IsSigner(addr common.Address) bool { return false }

func TestCreateTransactionSignUsesPoolNonce(t *testing.T) {
	password := "test-pass"
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)

	account, err := ks.ImportECDSA(acc1Key, password)
	if err != nil {
		t.Fatalf("failed to import signer account: %v", err)
	}
	if err := ks.Unlock(account, password); err != nil {
		t.Fatalf("failed to unlock signer account: %v", err)
	}

	manager := accounts.NewManager(nil, ks)
	defer manager.Close()

	subpool := &nonceGuardSubPool{}
	pool, err := txpool.New(0, createTxSignTestChain{}, []txpool.SubPool{subpool})
	if err != nil {
		t.Fatalf("failed to create txpool: %v", err)
	}
	defer pool.Close()

	chainConfig := params.TestXDPoSMockChainConfig
	if chainConfig == nil || chainConfig.XDPoS == nil {
		t.Fatal("test requires XDPoS chain config")
	}

	seedTx := CreateTxSign(big.NewInt(0), common.Hash{0x1}, 0, common.BlockSignersBinary)
	seedSigned, err := types.SignTx(seedTx, types.LatestSignerForChainID(chainConfig.ChainID), acc1Key)
	if err != nil {
		t.Fatalf("failed to sign seed tx: %v", err)
	}
	if err := pool.AddLocal(seedSigned, true); err != nil {
		t.Fatalf("failed to seed pending nonce 0 tx: %v", err)
	}

	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(0)})
	err = CreateTransactionSign(chainConfig, pool, manager, block, rawdb.NewMemoryDatabase(), account.Address)
	if errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Fatalf("CreateTransactionSign reused pending nonce and hit replacement rejection: %v", err)
	}
	if err != nil {
		t.Fatalf("CreateTransactionSign failed: %v", err)
	}

	if len(subpool.added) < 2 {
		t.Fatalf("expected seed tx and tx sign to be added, got %d txs", len(subpool.added))
	}
	if got := subpool.added[1].Nonce(); got != 1 {
		t.Fatalf("tx sign nonce mismatch: got %d, want 1", got)
	}
}
