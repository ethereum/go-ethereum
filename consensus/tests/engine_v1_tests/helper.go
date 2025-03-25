package engine_v1_tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	contractValidator "github.com/XinFinOrg/XDPoSChain/contracts/validator/contract"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

type masterNodes map[string]big.Int
type signersList map[string]bool

const GAP = int(450)

var (
	acc1Key, _  = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _  = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc3Key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	voterKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee04aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr    = crypto.PubkeyToAddress(acc1Key.PublicKey)  //xdc703c4b2bD70c169f5717101CaeE543299Fc946C7
	acc2Addr    = crypto.PubkeyToAddress(acc2Key.PublicKey)  //xdc0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e
	acc3Addr    = crypto.PubkeyToAddress(acc3Key.PublicKey)  //xdc71562b71999873DB5b286dF957af199Ec94617F7
	voterAddr   = crypto.PubkeyToAddress(voterKey.PublicKey) //xdc5F74529C0338546f82389402a01c31fB52c6f434
	chainID     = int64(1337)
)

func debugMessage(backend *backends.SimulatedBackend, signers signersList, t *testing.T) {
	ms := GetCandidateFromCurrentSmartContract(backend, t)
	fmt.Println("=== current smart contract")
	for nodeAddr, cap := range ms {
		if !strings.Contains(nodeAddr, "000000000000000000000000000000000000") { //remove defaults
			fmt.Println(nodeAddr, cap)
		}
	}
	fmt.Println("=== this block signer list")
	for signer := range signers {
		if !strings.Contains(signer, "000000000000000000000000000000000000") { //remove defaults
			fmt.Println(signer)
		}
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
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
		storage[key] = val
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

func transferTx(t *testing.T, to common.Address, transferAmount int64) *types.Transaction {
	t.Logf("Transfering %v to address: %v", transferAmount, to.String())
	data := []byte{}
	gasPrice := big.NewInt(int64(0))
	gasLimit := uint64(21000)
	amount := big.NewInt(transferAmount)
	nonce := uint64(1)
	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
	signedTX, err := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(chainID)), voterKey)
	if err != nil {
		t.Fatal(err)
	}
	return signedTX
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

// V1 consensus engine
func PrepareXDCTestBlockChain(t *testing.T, numOfBlocks int, chainConfig *params.ChainConfig) (*core.BlockChain, *backends.SimulatedBackend, *types.Block, common.Address, func(account accounts.Account, hash []byte) ([]byte, error)) {
	// Preparation
	var err error
	// Authorise
	signer, signFn, err := backends.SimulateWalletAddressAndSignFn()

	backend := getCommonBackend(t, chainConfig)
	blockchain := backend.BlockChain()
	blockchain.Client = backend

	if err != nil {
		panic(fmt.Errorf("error while creating simulated wallet for generating singer address and signer fn: %v", err))
	}
	blockchain.Engine().(*XDPoS.XDPoS).Authorize(signer, signFn)

	currentBlock := blockchain.Genesis()

	go func() {
		for range core.CheckpointCh {
			checkpointChanMsg := <-core.CheckpointCh
			log.Info("[V1] Got a message from core CheckpointChan!", "msg", checkpointChanMsg)
		}
	}()

	// Insert initial blocks
	for i := 1; i <= numOfBlocks; i++ {
		blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", i)
		merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
		header := &types.Header{
			Root:       common.HexToHash(merkleRoot),
			Number:     big.NewInt(int64(i)),
			ParentHash: currentBlock.Hash(),
			Coinbase:   common.HexToAddress(blockCoinBase),
		}
		block, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, chainConfig)
		if err != nil {
			t.Fatal(err)
		}
		err = blockchain.InsertBlock(block)
		if err != nil {
			panic(err)
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

func CreateBlock(blockchain *core.BlockChain, chainConfig *params.ChainConfig, startingBlock *types.Block, blockNumber int, roundNumber int64, blockCoinBase string, signer common.Address, signFn func(account accounts.Account, hash []byte) ([]byte, error), penalties []byte) *types.Block {
	currentBlock := startingBlock
	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"

	header := &types.Header{
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

	// TODO: check if this is needed
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
		Time:        big.NewInt(time.Now().Unix()),
		Extra:       customHeader.Extra,
		Validator:   customHeader.Validator,
		Validators:  customHeader.Validators,
		Penalties:   customHeader.Penalties,
	}
	var block *types.Block
	if len(txs) == 0 {
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
		block = types.NewBlock(&header, txs, nil, receipts, trie.NewStackTrie(nil))
	}

	return block, nil
}

// /*
// func proposeTX(t *testing.T) *types.Transaction {
// 	data := common.Hex2Bytes("012679510000000000000000000000000d3ab14bbad3d99f4203bd7a11acb94882050e7e")
// 	//data := []byte{}
// 	fmt.Println("data", string(data[:]))
// 	gasPrice := big.NewInt(int64(0))
// 	gasLimit := uint64(22680)
// 	amountInt := new(big.Int)
// 	amount, ok := amountInt.SetString("11000000000000000000000000", 10)
// 	if !ok {
// 		t.Fatal("big int init failed")
// 	}
// 	nonce := uint64(0)
// 	to := common.HexToAddress("xdc35658f7b2a9e7701e65e7a654659eb1c481d1dc5")
// 	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
// 	signedTX, err := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(chainID)), acc4Key)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	return signedTX
// }
// */
