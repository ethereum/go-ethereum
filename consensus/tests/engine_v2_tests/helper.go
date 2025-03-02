package engine_v2_tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	contractValidator "github.com/XinFinOrg/XDPoSChain/contracts/validator/contract"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/stretchr/testify/assert"
)

type masterNodes map[string]big.Int
type signersList map[string]bool

const GAP = int(450)

var (
	acc1Key, _  = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _  = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc3Key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc4Key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f292")
	acc5Key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f293")
	voterKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee04aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr    = crypto.PubkeyToAddress(acc1Key.PublicKey)  //xdc703c4b2bD70c169f5717101CaeE543299Fc946C7
	acc2Addr    = crypto.PubkeyToAddress(acc2Key.PublicKey)  //xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e
	acc3Addr    = crypto.PubkeyToAddress(acc3Key.PublicKey)  //xdc71562b71999873DB5b286dF957af199Ec94617F7
	voterAddr   = crypto.PubkeyToAddress(voterKey.PublicKey) //xdc5F74529C0338546f82389402a01c31fB52c6f434
	chainID     = int64(1337)

	protector1Key, _ = crypto.HexToECDSA("071c71a67e127fad4e901695e1b4b9ee04ae0e301d1e14474d32c45c72ce7b70")
	protector1Addr   = crypto.PubkeyToAddress(protector1Key.PublicKey)
	protector2Key, _ = crypto.HexToECDSA("1d1e144127fad4e9016a977b97b0c89921839df052d7adc2f789034678902378")
	protector2Addr   = crypto.PubkeyToAddress(protector2Key.PublicKey)
	observer1Key, _  = crypto.HexToECDSA("71a67e127fad4e9016a977b97b0c89921839df052d7adc2f7890346789023789")
	observer1Addr    = crypto.PubkeyToAddress(observer1Key.PublicKey)
	observer2Key, _  = crypto.HexToECDSA("789034678902378971a67e127fad4e9016a977b97b0c89921839df052d7adc2f")
	observer2Addr    = crypto.PubkeyToAddress(observer2Key.PublicKey)
)

func SignHashByPK(pk *ecdsa.PrivateKey, itemToSign []byte) []byte {
	signer, signFn, err := getSignerAndSignFn(pk)
	if err != nil {
		panic(err)
	}
	signedHash, err := signFn(accounts.Account{Address: signer}, itemToSign)
	if err != nil {
		panic(err)
	}
	return signedHash
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func getSignerAndSignFn(pk *ecdsa.PrivateKey) (common.Address, func(account accounts.Account, hash []byte) ([]byte, error), error) {
	veryLightScryptN := 2
	veryLightScryptP := 1
	dir, _ := os.MkdirTemp("", fmt.Sprintf("eth-getSignerAndSignFn-test-%v", RandStringBytes(5)))
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, veryLightScryptN, veryLightScryptP)
	pass := "" // not used but required by API
	a1, err := ks.ImportECDSA(pk, pass)
	if err != nil {
		return common.Address{}, nil, err
	}
	if err := ks.Unlock(a1, ""); err != nil {
		return a1.Address, nil, err
	}
	return a1.Address, ks.SignHash, nil
}

func voteTX(gasLimit uint64, nonce uint64, addr string) (*types.Transaction, error) {
	vote := "6dd7d8ea" // VoteMethod = "0x6dd7d8ea"
	action := fmt.Sprintf("%s%s%s", vote, "000000000000000000000000", addr[3:])
	data := common.Hex2Bytes(action)
	gasPrice := big.NewInt(int64(0))
	amountInt := new(big.Int)
	amount, ok := amountInt.SetString("60000", 10)
	if !ok {
		return nil, errors.New("big int init failed")
	}
	to := common.MasternodeVotingSMCBinary
	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)

	signedTX, err := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(chainID)), voterKey)
	if err != nil {
		return nil, err
	}

	return signedTX, nil
}

func getCommonBackend(t *testing.T, chainConfig *params.ChainConfig) *backends.SimulatedBackend {

	// initial helper backend
	contractBackendForSC := backends.NewXDCSimulatedBackend(types.GenesisAlloc{
		voterAddr: {Balance: new(big.Int).SetUint64(10000000000)},
	}, 10000000, chainConfig)

	transactOpts := bind.NewKeyedTransactor(voterKey)

	var candidates []common.Address
	var caps []*big.Int
	defalutCap := new(big.Int)
	defalutCap.SetString("1000000000", 10)

	for i := 1; i <= 16; i++ {
		addr := fmt.Sprintf("%02d", i)
		candidates = append(candidates, common.StringToAddress(addr)) // StringToAddress does not exist
		caps = append(caps, defalutCap)
	}

	acc1Cap, acc2Cap, acc3Cap, voterCap := new(big.Int), new(big.Int), new(big.Int), new(big.Int)

	acc1Cap.SetString("10000001", 10)
	acc2Cap.SetString("10000002", 10)
	acc3Cap.SetString("10000003", 10)
	voterCap.SetString("1000000000", 10)

	caps = append(caps, voterCap, acc1Cap, acc2Cap, acc3Cap)
	candidates = append(candidates, voterAddr, acc1Addr, acc2Addr, acc3Addr)
	// create validator smart contract
	validatorSCAddr, _, _, err := contractValidator.DeployXDCValidator(
		transactOpts,
		contractBackendForSC,
		candidates,
		caps,
		voterAddr, // first owner, not used
		big.NewInt(50000),
		big.NewInt(1),
		big.NewInt(99),
		big.NewInt(100),
		big.NewInt(100),
	)
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}

	contractBackendForSC.Commit() // Write into database(state)

	// Prepare Code and Storage
	d := time.Now().Add(1000 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	code, _ := contractBackendForSC.CodeAt(ctx, validatorSCAddr, nil)
	storage := make(map[common.Hash]common.Hash)
	f := func(key, val common.Hash) bool {
		decode := []byte{}
		trim := bytes.TrimLeft(val.Bytes(), "\x00")
		err := rlp.DecodeBytes(trim, &decode)
		if err != nil {
			t.Fatalf("Failed while decode byte")
		}
		storage[key] = common.BytesToHash(decode)
		log.Info("DecodeBytes", "value", val.String(), "decode", storage[key].String())
		return true
	}
	err = contractBackendForSC.ForEachStorageAt(ctx, validatorSCAddr, nil, f)
	if err != nil {
		t.Fatalf("Failed while trying to read all keys from SC")
	}

	// create test backend with smart contract in it
	contractBackend2 := backends.NewXDCSimulatedBackend(types.GenesisAlloc{
		acc1Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		acc2Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		acc3Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		voterAddr:                        {Balance: new(big.Int).SetUint64(10000000000)},
		common.MasternodeVotingSMCBinary: {Balance: new(big.Int).SetUint64(1), Code: code, Storage: storage}, // Binding the MasternodeVotingSMC with newly created 'code' for SC execution
	}, 10000000, chainConfig)

	return contractBackend2
}

func getMultiCandidatesBackend(t *testing.T, chainConfig *params.ChainConfig, n int) *backends.SimulatedBackend {
	assert.GreaterOrEqual(t, n, 4)
	// initial helper backend, give a very large gas limit
	contractBackendForSC := backends.NewXDCSimulatedBackend(types.GenesisAlloc{
		voterAddr: {Balance: new(big.Int).SetUint64(10000000000)},
	}, 1000000000, chainConfig)

	transactOpts := bind.NewKeyedTransactor(voterKey)

	var candidates []common.Address
	var caps []*big.Int
	defalutCap := new(big.Int)
	defalutCap.SetString("1000000000", 10)

	for i := 1; i <= n-4; i++ {
		addr := fmt.Sprintf("%04d", i)
		candidates = append(candidates, common.StringToAddress(addr))
		caps = append(caps, defalutCap)
	}

	acc1Cap, acc2Cap, acc3Cap, voterCap := new(big.Int), new(big.Int), new(big.Int), new(big.Int)

	acc1Cap.SetString("10000001", 10)
	acc2Cap.SetString("10000002", 10)
	acc3Cap.SetString("10000003", 10)
	voterCap.SetString("2000000000", 10) // give voter the highest cap to make it win the masternode selection

	caps = append(caps, voterCap, acc1Cap, acc2Cap, acc3Cap)
	candidates = append(candidates, voterAddr, acc1Addr, acc2Addr, acc3Addr)

	// create validator smart contract
	validatorSCAddr, _, _, err := contractValidator.DeployXDCValidator(
		transactOpts,
		contractBackendForSC,
		candidates,
		caps,
		voterAddr, // first owner, not used
		big.NewInt(50000),
		big.NewInt(1),
		big.NewInt(99),
		big.NewInt(100),
		big.NewInt(100),
	)
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}

	contractBackendForSC.Commit() // Write into database(state)

	// Prepare Code and Storage
	d := time.Now().Add(3000 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	code, _ := contractBackendForSC.CodeAt(ctx, validatorSCAddr, nil)
	storage := make(map[common.Hash]common.Hash)
	f := func(key, val common.Hash) bool {
		decode := []byte{}
		trim := bytes.TrimLeft(val.Bytes(), "\x00")
		err := rlp.DecodeBytes(trim, &decode)
		if err != nil {
			t.Fatalf("Failed while decode byte")
		}
		storage[key] = common.BytesToHash(decode)
		log.Info("DecodeBytes", "value", val.String(), "decode", storage[key].String())
		return true
	}
	err = contractBackendForSC.ForEachStorageAt(ctx, validatorSCAddr, nil, f)
	if err != nil {
		t.Fatalf("Failed while trying to read all keys from SC")
	}

	// create test backend with smart contract in it
	contractBackend2 := backends.NewXDCSimulatedBackend(types.GenesisAlloc{
		acc1Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		acc2Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		acc3Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		voterAddr:                        {Balance: new(big.Int).SetUint64(10000000000)},
		common.MasternodeVotingSMCBinary: {Balance: new(big.Int).SetUint64(1), Code: code, Storage: storage}, // Binding the MasternodeVotingSMC with newly created 'code' for SC execution
	}, 10000000, chainConfig)

	return contractBackend2
}

func getProtectorObserverBackend(t *testing.T, chainConfig *params.ChainConfig) *backends.SimulatedBackend {

	// initial helper backend
	contractBackendForSC := backends.NewXDCSimulatedBackend(types.GenesisAlloc{
		voterAddr: {Balance: new(big.Int).SetUint64(10000000000)},
	}, 10000000, chainConfig)

	transactOpts := bind.NewKeyedTransactor(voterKey)

	var candidates []common.Address
	var caps []*big.Int
	defalutCap := new(big.Int)
	defalutCap.SetString("1000000000", 10)

	for i := 1; i <= 15; i++ {
		addr := fmt.Sprintf("%02d", i)
		candidates = append(candidates, common.StringToAddress(addr)) // StringToAddress does not exist
		caps = append(caps, defalutCap)
	}
	candidates = append(candidates, protector1Addr, protector2Addr, observer1Addr, observer2Addr)
	caps = append(caps, defalutCap, defalutCap, big.NewInt(999999), big.NewInt(999999)) // 99..9 is a small cap

	acc1Cap, acc2Cap, acc3Cap, voterCap := new(big.Int), new(big.Int), new(big.Int), new(big.Int)

	acc1Cap.SetString("10000001", 10)
	acc2Cap.SetString("10000002", 10)
	acc3Cap.SetString("10000003", 10)
	voterCap.SetString("1000000000", 10)

	caps = append(caps, voterCap, acc1Cap, acc2Cap, acc3Cap)
	candidates = append(candidates, voterAddr, acc1Addr, acc2Addr, acc3Addr)
	// create validator smart contract
	validatorSCAddr, _, _, err := contractValidator.DeployXDCValidator(
		transactOpts,
		contractBackendForSC,
		candidates,
		caps,
		voterAddr, // first owner, not used
		big.NewInt(50000),
		big.NewInt(1),
		big.NewInt(99),
		big.NewInt(100),
		big.NewInt(100),
	)
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}

	contractBackendForSC.Commit() // Write into database(state)

	// Prepare Code and Storage
	d := time.Now().Add(1000 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	code, _ := contractBackendForSC.CodeAt(ctx, validatorSCAddr, nil)
	storage := make(map[common.Hash]common.Hash)
	f := func(key, val common.Hash) bool {
		decode := []byte{}
		trim := bytes.TrimLeft(val.Bytes(), "\x00")
		err := rlp.DecodeBytes(trim, &decode)
		if err != nil {
			t.Fatalf("Failed while decode byte")
		}
		storage[key] = common.BytesToHash(decode)
		log.Info("DecodeBytes", "value", val.String(), "decode", storage[key].String())
		return true
	}
	err = contractBackendForSC.ForEachStorageAt(ctx, validatorSCAddr, nil, f)
	if err != nil {
		t.Fatalf("Failed while trying to read all keys from SC")
	}

	// create test backend with smart contract in it
	contractBackend2 := backends.NewXDCSimulatedBackend(types.GenesisAlloc{
		acc1Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		acc2Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		acc3Addr:                         {Balance: new(big.Int).SetUint64(10000000000)},
		voterAddr:                        {Balance: new(big.Int).SetUint64(10000000000)},
		common.MasternodeVotingSMCBinary: {Balance: new(big.Int).SetUint64(1), Code: code, Storage: storage}, // Binding the MasternodeVotingSMC with newly created 'code' for SC execution
	}, 10000000, chainConfig)

	return contractBackend2
}

func signingTxWithKey(header *types.Header, nonce uint64, privateKey *ecdsa.PrivateKey) (*types.Transaction, error) {
	tx := contracts.CreateTxSign(header.Number, header.Hash(), nonce, common.BlockSignersBinary)
	s := types.LatestSignerForChainID(big.NewInt(chainID))
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], privateKey)
	if err != nil {
		return nil, err
	}
	signedTx, err := tx.WithSignature(s, sig)
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}

func signingTxWithSignerFn(header *types.Header, nonce uint64, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error)) (*types.Transaction, error) {
	tx := contracts.CreateTxSign(header.Number, header.Hash(), nonce, common.BlockSignersBinary)
	s := types.LatestSignerForChainID(big.NewInt(chainID))
	h := s.Hash(tx)
	sig, err := signFn(accounts.Account{Address: signer}, h[:])
	if err != nil {
		return nil, err
	}
	signedTx, err := tx.WithSignature(s, sig)
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}

func UpdateSigner(bc *core.BlockChain) error {
	err := bc.UpdateM1()
	return err
}

func GetSnapshotSigner(bc *core.BlockChain, header *types.Header) (signersList, error) {
	engine := bc.Engine().(*XDPoS.XDPoS)
	snap, err := engine.GetSnapshot(bc, header)
	if err != nil {
		return nil, err

	}
	ms := make(signersList)

	for addr := range snap.Signers {
		ms[addr.Hex()] = true
	}
	return ms, nil

}

func GetCandidateFromCurrentSmartContract(backend bind.ContractBackend, t *testing.T) masterNodes {
	addr := common.MasternodeVotingSMCBinary
	validator, err := contractValidator.NewXDCValidator(addr, backend)
	if err != nil {
		t.Fatal(err)
	}

	opts := new(bind.CallOpts)
	candidates, err := validator.GetCandidates(opts)
	if err != nil {
		t.Fatal(err)
	}

	ms := make(masterNodes)
	for _, candidate := range candidates {
		v, err := validator.GetCandidateCap(opts, candidate)
		if err != nil {
			t.Fatal(err)
		}
		ms[candidate.String()] = *v
	}
	return ms
}

type ForkedBlockOptions struct {
	numOfForkedBlocks     *int
	forkedRoundDifference *int // Minimum is 1
	signersKey            []*ecdsa.PrivateKey
}

// V2 concensus engine
func PrepareXDCTestBlockChainForV2Engine(t *testing.T, numOfBlocks int, chainConfig *params.ChainConfig, forkedBlockOptions *ForkedBlockOptions) (*core.BlockChain, *backends.SimulatedBackend, *types.Block, common.Address, func(account accounts.Account, hash []byte) ([]byte, error), *types.Block) {
	// Preparation
	var err error
	signer, signFn, err := backends.SimulateWalletAddressAndSignFn()
	if err != nil {
		panic(fmt.Errorf("error while creating simulated wallet for generating singer address and signer fn: %v", err))
	}
	backend := getCommonBackend(t, chainConfig)
	blockchain := backend.BlockChain()
	blockchain.Client = backend

	engine := blockchain.Engine().(*XDPoS.XDPoS)

	// Authorise
	engine.Authorize(signer, signFn)

	currentBlock := blockchain.Genesis()

	var currentForkBlock *types.Block

	go func() {
		for range core.CheckpointCh {
			checkpointChanMsg := <-core.CheckpointCh
			log.Info("[V2] Got a message from core CheckpointChan!", "msg", checkpointChanMsg)
		}
	}()

	// Insert initial blocks
	for i := 1; i <= numOfBlocks; i++ {
		blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", i)
		// for v2 blocks, fill in correct coinbase
		if int64(i) > chainConfig.XDPoS.V2.SwitchBlock.Int64() {
			blockCoinBase = signer.Hex()
		}
		roundNumber := int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()
		block := CreateBlock(blockchain, chainConfig, currentBlock, i, roundNumber, blockCoinBase, signer, signFn, nil, nil, "")

		err = blockchain.InsertBlock(block)
		if err != nil {
			t.Fatal(err)
		}
		// Produce forked block for the last numOfForkedBlocks'th blocks
		if forkedBlockOptions != nil && forkedBlockOptions.numOfForkedBlocks != nil && i > numOfBlocks-*forkedBlockOptions.numOfForkedBlocks {
			if currentForkBlock == nil {
				currentForkBlock = currentBlock
			}
			forkedBlockCoinBase := fmt.Sprintf("0x222000000000000000000000000000000%03d", i)
			var forkedBlockRoundNumber int64
			if forkedBlockOptions.forkedRoundDifference != nil {
				if *forkedBlockOptions.forkedRoundDifference == 0 {
					t.Fatal("forkedRoundDifference minimum is 1")
				}
				forkedBlockRoundNumber = roundNumber + int64(*forkedBlockOptions.forkedRoundDifference)
			} else {
				forkedBlockRoundNumber = roundNumber + int64(*forkedBlockOptions.numOfForkedBlocks)
			}

			forkedBlock := CreateBlock(blockchain, chainConfig, currentForkBlock, i, forkedBlockRoundNumber, forkedBlockCoinBase, signer, signFn, nil, forkedBlockOptions.signersKey, "")

			err = blockchain.InsertBlock(forkedBlock)
			if err != nil {
				panic(err)
			}
			currentForkBlock = forkedBlock
		}

		// First v2 block
		if (int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()) == 1 {
			lastv1BlockNumber := block.Header().Number.Uint64() - 1
			checkpointBlockNumber := lastv1BlockNumber - lastv1BlockNumber%chainConfig.XDPoS.Epoch
			checkpointHeader := blockchain.GetHeaderByNumber(checkpointBlockNumber)
			err := engine.EngineV2.Initial(blockchain, checkpointHeader)
			if err != nil {
				panic(err)
			}
		}

		currentBlock = block
	}

	// Update Signer as there is no previous signer assigned
	err = UpdateSigner(blockchain)
	if err != nil {
		t.Fatal(err)
	}

	return blockchain, backend, currentBlock, signer, signFn, currentForkBlock
}

// V2 concensus engine, compared to PrepareXDCTestBlockChainForV2Engine: (1) no forking (2) add penalty
func PrepareXDCTestBlockChainWithPenaltyForV2Engine(t *testing.T, numOfBlocks int, chainConfig *params.ChainConfig) (*core.BlockChain, *backends.SimulatedBackend, *types.Block, common.Address, func(account accounts.Account, hash []byte) ([]byte, error)) {
	// Preparation
	var err error
	signer, signFn, err := backends.SimulateWalletAddressAndSignFn()
	if err != nil {
		t.Fatal("Error while creating simulated wallet for generating singer address and signer fn: ", err)
	}
	backend := getCommonBackend(t, chainConfig)
	blockchain := backend.BlockChain()
	blockchain.Client = backend

	// Authorise
	blockchain.Engine().(*XDPoS.XDPoS).Authorize(signer, signFn)

	currentBlock := blockchain.Genesis()

	go func() {
		for range core.CheckpointCh {
			checkpointChanMsg := <-core.CheckpointCh
			log.Info("[V2] Got a message from core CheckpointChan!", "msg", checkpointChanMsg)
		}
	}()

	// Insert initial blocks
	for i := 1; i <= numOfBlocks; i++ {
		blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", i)
		// for v2 blocks, fill in correct coinbase
		if int64(i) > chainConfig.XDPoS.V2.SwitchBlock.Int64() {
			blockCoinBase = signer.Hex()
		}
		roundNumber := int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()
		// use signer itself as penalty
		penalty := signer[:]
		if roundNumber%int64(chainConfig.XDPoS.Epoch) != 0 {
			penalty = nil
		}
		block := CreateBlock(blockchain, chainConfig, currentBlock, i, roundNumber, blockCoinBase, signer, signFn, penalty, nil, "")

		err = blockchain.InsertBlock(block)
		if err != nil {
			t.Fatal(err)
		}
		currentBlock = block
	}

	// Update Signer as there is no previous signer assigned
	err = UpdateSigner(blockchain)
	if err != nil {
		t.Fatal(err)
	}

	return blockchain, backend, currentBlock, signer, signFn
}

// V2 concensus engine, compared to PrepareXDCTestBlockChainForV2Engine: (1) no forking (2) 128 masternode candidates
func PrepareXDCTestBlockChainWith128Candidates(t *testing.T, numOfBlocks int, chainConfig *params.ChainConfig) (*core.BlockChain, *backends.SimulatedBackend, *types.Block, common.Address, func(account accounts.Account, hash []byte) ([]byte, error)) {
	// Preparation
	var err error
	signer, signFn, err := backends.SimulateWalletAddressAndSignFn()
	if err != nil {
		t.Fatal("Error while creating simulated wallet for generating singer address and signer fn: ", err)
	}
	backend := getMultiCandidatesBackend(t, chainConfig, 128)
	blockchain := backend.BlockChain()
	blockchain.Client = backend

	engine := blockchain.Engine().(*XDPoS.XDPoS)

	// Authorise
	engine.Authorize(signer, signFn)

	currentBlock := blockchain.Genesis()

	go func() {
		for range core.CheckpointCh {
			checkpointChanMsg := <-core.CheckpointCh
			log.Info("[V2] Got a message from core CheckpointChan!", "msg", checkpointChanMsg)
		}
	}()

	// Insert initial blocks
	for i := 1; i <= numOfBlocks; i++ {
		blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", i)
		// for v2 blocks, fill in correct coinbase
		if int64(i) > chainConfig.XDPoS.V2.SwitchBlock.Int64() {
			blockCoinBase = signer.Hex()
		}
		roundNumber := int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()
		block := CreateBlock(blockchain, chainConfig, currentBlock, i, roundNumber, blockCoinBase, signer, signFn, nil, nil, "b345a8560bd51926803dd17677c9f0751193914a851a4ec13063d6bf50220b53")
		err = blockchain.InsertBlock(block)
		if err != nil {
			t.Fatal(err)
		}

		// First v2 block
		if (int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()) == 1 {
			lastv1BlockNumber := block.Header().Number.Uint64() - 1
			checkpointBlockNumber := lastv1BlockNumber - lastv1BlockNumber%chainConfig.XDPoS.Epoch
			checkpointHeader := blockchain.GetHeaderByNumber(checkpointBlockNumber)
			err := engine.EngineV2.Initial(blockchain, checkpointHeader)
			if err != nil {
				panic(err)
			}
		}

		currentBlock = block
	}

	// Update Signer as there is no previous signer assigned
	err = UpdateSigner(blockchain)
	if err != nil {
		t.Fatal(err)
	}

	return blockchain, backend, currentBlock, signer, signFn
}

// V2 concensus engine
func PrepareXDCTestBlockChainWithProtectorObserver(t *testing.T, numOfBlocks int, chainConfig *params.ChainConfig) (*core.BlockChain, *backends.SimulatedBackend, *types.Block, common.Address, func(account accounts.Account, hash []byte) ([]byte, error)) {
	// Preparation
	var err error
	signer, signFn, err := backends.SimulateWalletAddressAndSignFn()
	if err != nil {
		panic(fmt.Errorf("error while creating simulated wallet for generating singer address and signer fn: %v", err))
	}
	backend := getProtectorObserverBackend(t, chainConfig)
	blockchain := backend.BlockChain()
	blockchain.Client = backend

	engine := blockchain.Engine().(*XDPoS.XDPoS)

	// Authorise
	engine.Authorize(signer, signFn)

	currentBlock := blockchain.Genesis()

	go func() {
		for range core.CheckpointCh {
			checkpointChanMsg := <-core.CheckpointCh
			log.Info("[V2] Got a message from core CheckpointChan!", "msg", checkpointChanMsg)
		}
	}()

	// Insert initial blocks
	for i := 1; i <= numOfBlocks; i++ {
		blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", i)
		// for v2 blocks, fill in correct coinbase
		if int64(i) > chainConfig.XDPoS.V2.SwitchBlock.Int64() {
			blockCoinBase = signer.Hex()
		}
		roundNumber := int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()
		// use observer2 as penalty and put in checkpoint block
		penalty := observer2Addr[:]
		if i != 900 {
			penalty = nil
		}
		block := CreateBlock(blockchain, chainConfig, currentBlock, i, roundNumber, blockCoinBase, signer, signFn, penalty, nil, "f11ec19df702aa6bd9b3b2186edbc66d6b50b06334455a4a2ae8d166f28a14ff")

		if i == 900 {
			fmt.Println(block.Penalties())
		}
		err = blockchain.InsertBlock(block)
		if err != nil {
			t.Fatal(err)
		}

		// First v2 block
		if (int64(i) - chainConfig.XDPoS.V2.SwitchBlock.Int64()) == 1 {
			lastv1BlockNumber := block.Header().Number.Uint64() - 1
			checkpointBlockNumber := lastv1BlockNumber - lastv1BlockNumber%chainConfig.XDPoS.Epoch
			checkpointHeader := blockchain.GetHeaderByNumber(checkpointBlockNumber)
			err := engine.EngineV2.Initial(blockchain, checkpointHeader)
			if err != nil {
				panic(err)
			}
		}

		currentBlock = block
	}

	// Update Signer as there is no previous signer assigned
	err = UpdateSigner(blockchain)
	if err != nil {
		t.Fatal(err)
	}

	return blockchain, backend, currentBlock, signer, signFn
}

func CreateBlock(blockchain *core.BlockChain, chainConfig *params.ChainConfig, startingBlock *types.Block, blockNumber int, roundNumber int64, blockCoinBase string, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error), penalties []byte, signersKey []*ecdsa.PrivateKey, merkleRoot string) *types.Block {
	currentBlock := startingBlock
	if len(merkleRoot) == 0 {
		merkleRoot = "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	}
	var header *types.Header

	if big.NewInt(int64(blockNumber)).Cmp(chainConfig.XDPoS.V2.SwitchBlock) == 1 { // Build engine v2 compatible extra data field
		extraInBytes := generateV2Extra(roundNumber, currentBlock, signer, signFn, signersKey)

		header = &types.Header{
			Root:       common.HexToHash(merkleRoot),
			Number:     big.NewInt(int64(blockNumber)),
			ParentHash: currentBlock.Hash(),
			Coinbase:   common.HexToAddress(blockCoinBase),
			Extra:      extraInBytes,
		}
		if int64(blockNumber) == (chainConfig.XDPoS.V2.SwitchBlock.Int64() + 1) { // This is the first v2 block, we need to copy the last v1 epoch master node list and inject into v2 validators
			// Get last master node list from last v1 block
			lastv1Block := blockchain.GetBlockByNumber(chainConfig.XDPoS.V2.SwitchBlock.Uint64())
			masternodesFromV1LastEpoch := decodeMasternodesFromHeaderExtra(lastv1Block.Header())
			for _, v := range masternodesFromV1LastEpoch {
				header.Validators = append(header.Validators, v[:]...)
			}
		} else if roundNumber%int64(chainConfig.XDPoS.Epoch) == 0 {
			// epoch switch blocks, copy the master node list and inject into v2 validators
			// Get last master node list from last v1 block
			lastv1Block := blockchain.GetBlockByNumber(chainConfig.XDPoS.V2.SwitchBlock.Uint64())
			masternodesFromV1LastEpoch := decodeMasternodesFromHeaderExtra(lastv1Block.Header())
			for _, v := range masternodesFromV1LastEpoch {
				header.Validators = append(header.Validators, v[:]...)
			}
		}
	} else {
		// V1 block
		header = &types.Header{
			Root:       common.HexToHash(merkleRoot),
			Number:     big.NewInt(int64(blockNumber)),
			ParentHash: currentBlock.Hash(),
			Coinbase:   common.HexToAddress(blockCoinBase),
		}

		// Inject the hardcoded master node list for the last v1 epoch block and all v1 epoch switch blocks (excluding genesis)
		if big.NewInt(int64(blockNumber)).Cmp(chainConfig.XDPoS.V2.SwitchBlock) == 0 || blockNumber%int(chainConfig.XDPoS.Epoch) == 0 {
			// reset extra
			header.Extra = []byte{}
			if len(header.Extra) < utils.ExtraVanity {
				header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, utils.ExtraVanity-len(header.Extra))...)
			}
			header.Extra = header.Extra[:utils.ExtraVanity]
			var masternodes []common.Address
			// Place the test's signer address to the last
			masternodes = append(masternodes, acc1Addr, acc2Addr, acc3Addr, voterAddr, signer)
			// masternodesFromV1LastEpoch = masternodes
			for _, masternode := range masternodes {
				header.Extra = append(header.Extra, masternode[:]...)
			}
			header.Extra = append(header.Extra, make([]byte, utils.ExtraSeal)...)

			// Sign all the things for v1 block use v1 sigHash function
			sighash, err := signFn(accounts.Account{Address: signer}, blockchain.Engine().(*XDPoS.XDPoS).SigHash(header).Bytes())
			if err != nil {
				panic(errors.New("error when sign last v1 block hash during test block creation"))
			}
			copy(header.Extra[len(header.Extra)-utils.ExtraSeal:], sighash)
		}
	}
	if penalties != nil {
		header.Penalties = penalties
	}
	block, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, chainConfig)
	if err != nil {
		panic(fmt.Errorf("fail to create block in test helper, %v", err))
	}
	return block
}

func createBlockFromHeader(bc *core.BlockChain, customHeader *types.Header, txs []*types.Transaction, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error), config *params.ChainConfig) (*types.Block, error) {
	if customHeader.Extra == nil {
		extraSubstring := "d7830100018358444388676f312e31342e31856c696e75780000000000000000b185dc0d0e917d18e5dbf0746be6597d3331dd27ea0554e6db433feb2e81730b20b2807d33a1527bf43cd3bc057aa7f641609c2551ebe2fd575f4db704fbf38101" // Grabbed from existing mainnet block, it does not have any meaning except for the length validation
		customHeader.Extra, _ = hex.DecodeString(extraSubstring)
	}
	var difficulty *big.Int
	if customHeader.Difficulty == nil {
		difficulty = big.NewInt(1)
	} else {
		difficulty = customHeader.Difficulty
	}

	if len(txs) != 0 {
		customHeader.ReceiptHash = common.HexToHash("0x9319777b782ba2c83a33c995481ff894ac96d9a92a1963091346a3e1e386705c")
	} else {
		customHeader.ReceiptHash = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	}

	header := types.Header{
		ParentHash:  customHeader.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: customHeader.ReceiptHash,
		Root:        customHeader.Root,
		Coinbase:    customHeader.Coinbase,
		Difficulty:  difficulty,
		Number:      customHeader.Number,
		GasLimit:    1200000000,
		Time:        big.NewInt(time.Now().Unix() - 1000000 + int64(customHeader.Number.Uint64()*10)),
		Extra:       customHeader.Extra,
		Validator:   customHeader.Validator,
		Validators:  customHeader.Validators,
		Penalties:   customHeader.Penalties,
	}
	var block *types.Block
	if len(txs) == 0 {
		// Sign all the things and seal it
		signerAddress, signerFunction := findSignerAndSignFn(bc, &header, signer, signFn, config)
		header.Coinbase = signerAddress
		sealHeader(bc, &header, signerAddress, signerFunction)

		block = types.NewBlockWithHeader(&header)
	} else {
		// Prepare Receipt
		statedb, err := bc.StateAt(bc.GetBlockByNumber(customHeader.Number.Uint64() - 1).Root()) //Get parent root
		if err != nil {
			return nil, fmt.Errorf("%v when get state", err)
		}
		gp := new(core.GasPool).AddGas(header.GasLimit)

		var gasUsed = new(uint64)
		var receipts types.Receipts
		for i, tx := range txs {
			statedb.SetTxContext(tx.Hash(), i)
			receipt, _, err, _ := core.ApplyTransaction(bc.Config(), nil, bc, &header.Coinbase, gp, statedb, nil, &header, tx, gasUsed, vm.Config{})
			if err != nil {
				return nil, fmt.Errorf("%v when applying transaction", err)
			}
			receipts = append(receipts, receipt)
		}

		header.GasUsed = *gasUsed

		// Sign all the things and seal it
		signerAddress, signerFunction := findSignerAndSignFn(bc, &header, signer, signFn, config)
		header.Coinbase = signerAddress
		sealHeader(bc, &header, signerAddress, signerFunction)

		block = types.NewBlock(&header, txs, nil, receipts)
	}

	return block, nil
}

// Get masternodes address from checkpoint Header. Only used for v1 last block
func decodeMasternodesFromHeaderExtra(checkpointHeader *types.Header) []common.Address {
	masternodes := make([]common.Address, (len(checkpointHeader.Extra)-utils.ExtraVanity-utils.ExtraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], checkpointHeader.Extra[utils.ExtraVanity+i*common.AddressLength:])
	}
	return masternodes
}

func findSignerAndSignFn(bc *core.BlockChain, header *types.Header, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error), config *params.ChainConfig) (common.Address, func(account accounts.Account, hash []byte) ([]byte, error)) {
	addressToSign := signer
	addressedSignFn := signFn

	// If v2 block, we need to use extra data's round to find who is creating the block in order to verify the validator
	if header.Number.Cmp(config.XDPoS.V2.SwitchBlock) > 0 {
		var decodedExtraField types.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(header.Extra, &decodedExtraField)
		if err != nil {
			panic(errors.New("fail to seal header for v2 block"))
		}
		round := decodedExtraField.Round
		masterNodes := getMasternodesList(signer)

		index := uint64(round) % config.XDPoS.Epoch % uint64(len(masterNodes))
		// index 0 to 2 are acc1Addr, acc2Addr, acc3Addr
		addressToSign = masterNodes[index]
		if index == 0 {
			_, signFn, err = getSignerAndSignFn(acc1Key)
		} else if index == 1 {
			_, signFn, err = getSignerAndSignFn(acc2Key)
		} else if index == 2 {
			_, signFn, err = getSignerAndSignFn(acc3Key)
		} else if index == 3 {
			// Skip signing anything for voterAddress to simulate penalty
			return signer, signFn
		}
		addressedSignFn = signFn
		if err != nil {
			panic(errors.New("error trying to use one of the pre-defined private key to sign"))
		}
	}

	return addressToSign, addressedSignFn
}

func sealHeader(bc *core.BlockChain, header *types.Header, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error)) {
	// Sign all the things and seal it
	signedBlockHeader := bc.Engine().(*XDPoS.XDPoS).SigHash(header)

	signature, err := signFn(accounts.Account{Address: signer}, signedBlockHeader.Bytes())
	if err != nil {
		panic(err)
	}
	header.Validator = signature
}

func getMasternodesList(signer common.Address) []common.Address {
	var masternodes []common.Address
	// Place the test's signer address to the last
	masternodes = append(masternodes, acc1Addr, acc2Addr, acc3Addr, voterAddr, signer)
	return masternodes
}

func generateV2Extra(roundNumber int64, currentBlock *types.Block, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error), accKeys []*ecdsa.PrivateKey) []byte {
	var extraField types.ExtraFields_v2
	var round types.Round
	err := utils.DecodeBytesExtraFields(currentBlock.Extra(), &extraField)
	if err != nil {
		round = types.Round(0)
	} else {
		round = extraField.Round
	}

	proposedBlockInfo := &types.BlockInfo{
		Hash:   currentBlock.Hash(),
		Round:  round,
		Number: currentBlock.Number(),
	}
	gapNumber := currentBlock.Number().Uint64() - currentBlock.Number().Uint64()%params.TestXDPoSMockChainConfig.XDPoS.Epoch - params.TestXDPoSMockChainConfig.XDPoS.Gap
	voteForSign := &types.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         gapNumber,
	}

	signedHash, err := signFn(accounts.Account{Address: signer}, types.VoteSigHash(voteForSign).Bytes())
	if err != nil {
		panic(fmt.Errorf("error generate QC by creating signedHash: %v", err))
	}
	var signatures []types.Signature
	if len(accKeys) == 0 {
		// Sign from acc 1, 2, 3 by default
		accKeys = append(accKeys, acc1Key, acc2Key, acc3Key, voterKey)
	}
	for _, acc := range accKeys {
		h := SignHashByPK(acc, types.VoteSigHash(voteForSign).Bytes())
		signatures = append(signatures, h)
	}
	signatures = append(signatures, signedHash)

	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        signatures,
		GapNumber:         gapNumber,
	}

	extra := types.ExtraFields_v2{
		Round:      types.Round(roundNumber),
		QuorumCert: quorumCert,
	}
	extraInBytes, err := extra.EncodeToBytes()
	if err != nil {
		panic(fmt.Errorf("error encode extra into bytes: %v", err))
	}
	return extraInBytes
}
