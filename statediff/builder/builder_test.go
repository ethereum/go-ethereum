package builder_test

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	b "github.com/ethereum/go-ethereum/statediff/builder"
	"math/big"
	"reflect"
	"testing"
)

var (
	testdb = ethdb.NewMemDatabase()

	testBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey) //0x71562b71999873DB5b286dF957af199Ec94617F7
	testBankFunds   = big.NewInt(100000000)
	genesis         = core.GenesisBlockForTesting(testdb, testBankAddress, testBankFunds)

	account1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	account2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	account1Addr   = crypto.PubkeyToAddress(account1Key.PublicKey) //0x703c4b2bD70c169f5717101CaeE543299Fc946C7
	account2Addr   = crypto.PubkeyToAddress(account2Key.PublicKey) //0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e
	contractCode = common.Hex2Bytes("608060405234801561001057600080fd5b50602060405190810160405280600160ff16815250600090600161003592919061003b565b506100a5565b826064810192821561006f579160200282015b8281111561006e578251829060ff1690559160200191906001019061004e565b5b50905061007c9190610080565b5090565b6100a291905b8082111561009e576000816000905550600101610086565b5090565b90565b610124806100b46000396000f3fe6080604052348015600f57600080fd5b5060043610604f576000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146054578063c16431b9146093575b600080fd5b607d60048036036020811015606857600080fd5b810190808035906020019092919050505060c8565b6040518082815260200191505060405180910390f35b60c66004803603604081101560a757600080fd5b81019080803590602001909291908035906020019092919050505060e0565b005b6000808260648110151560d757fe5b01549050919050565b8060008360648110151560ef57fe5b0181905550505056fea165627a7a7230582064e918c3140a117bf3aa65865a9b9e83fae21ad1720506e7933b2a9f54bb40260029")
	contractAddr common.Address
	emptyAccountDiffEventualMap    = make(map[common.Address]b.AccountDiff)
	emptyAccountDiffIncrementalMap = make(map[common.Address]b.AccountDiff)
	block0Hash, block1Hash, block2Hash, block3Hash common.Hash
	block0, block1, block2, block3                 *types.Block
	builder                                        b.Builder
	miningReward                                   = int64(2000000000000000000)
	burnAddress                                    = common.HexToAddress("0x0")
)

func TestBuilder(t *testing.T) {
	_, blockMap := makeChain(3, genesis)
	block0Hash = common.HexToHash("0xd1721cfd0b29c36fd7a68f25c128e86413fb666a6e1d68e89b875bd299262661")
	block1Hash = common.HexToHash("0xbbe88de60ba33a3f18c0caa37d827bfb70252e19e40a07cd34041696c35ecb1a")
	block2Hash = common.HexToHash("0xde75663f36a8497b4bdda2a4b52bd9540b705a2728c7391c59b8cb2cde5a2feb")
	block3Hash = common.HexToHash("0x76c6d0e39285cee40d5e5fadc6141ca88c8ab8bd1a15d46717205af2efbb4a3c")

	block0 = blockMap[block0Hash]
	block1 = blockMap[block1Hash]
	block2 = blockMap[block2Hash]
	block3 = blockMap[block3Hash]
	builder = b.NewBuilder(testdb)

	type arguments struct {
		oldStateRoot common.Hash
		newStateRoot common.Hash
		blockNumber  int64
		blockHash    common.Hash
	}

	var (
		balanceChange10000      = int64(10000)
		balanceChange1000       = int64(1000)
		block1BankBalance       = int64(99990000)
		block1Account1Balance   = int64(10000)
		block2Account2Balance   = int64(1000)
		nonce0                  = uint64(0)
		nonce1                  = uint64(1)
		nonce2                  = uint64(2)
		nonce3                  = uint64(3)
		originalContractRoot    = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
		contractContractRoot    = "0x821e2556a290c86405f8160a2d662042a431ba456b9db265c79bb837c04be5f0"
		newContractRoot         = "0x71e0d14b2b93e5c7f9748e69e1fe5f17498a1c3ac3cec29f96af13d7f8a4e070"
		originalStorageLocation = common.HexToHash("0")
		originalStorageKey      = crypto.Keccak256Hash(originalStorageLocation[:]).String()
		updatedStorageLocation = common.HexToHash("2")
		updatedStorageKey      = crypto.Keccak256Hash(updatedStorageLocation[:]).String()
		originalStorageValue = "0x01"
		updatedStorageValue  = "0x03"
	)

	var tests = []struct {
		name              string
		startingArguments arguments
		expected          *b.StateDiff
	}{
		{
			"testEmptyDiff",
			arguments{
				oldStateRoot: block0.Root(),
				newStateRoot: block0.Root(),
				blockNumber:  block0.Number().Int64(),
				blockHash:    block0Hash,
			},
			&b.StateDiff{
				BlockNumber:     block0.Number().Int64(),
				BlockHash:       block0Hash,
				CreatedAccounts: emptyAccountDiffEventualMap,
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: emptyAccountDiffIncrementalMap,
			},
		},
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			arguments{
				oldStateRoot: block0.Root(),
				newStateRoot: block1.Root(),
				blockNumber:  block1.Number().Int64(),
				blockHash:    block1Hash,
			},
			&b.StateDiff{
				BlockNumber: block1.Number().Int64(),
				BlockHash:   block1.Hash(),
				CreatedAccounts: map[common.Address]b.AccountDiff{
					account1Addr: {
						Nonce:        b.DiffUint64{Value: &nonce0},
						Balance:      b.DiffBigInt{Value: big.NewInt(balanceChange10000)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
					burnAddress: {
						Nonce:        b.DiffUint64{Value: &nonce0},
						Balance:      b.DiffBigInt{Value: big.NewInt(miningReward)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
				},
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: map[common.Address]b.AccountDiff{
					testBankAddress: {
						Nonce:        b.DiffUint64{Value: &nonce1},
						Balance:      b.DiffBigInt{Value: big.NewInt(testBankFunds.Int64() - balanceChange10000)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
				},
			},
		},
		{
			"testBlock2",
			//1000 transferred from testBankAddress to account1Addr
			//1000 transferred from account1Addr to account2Addr
			arguments{
				oldStateRoot: block1.Root(),
				newStateRoot: block2.Root(),
				blockNumber:  block2.Number().Int64(),
				blockHash:    block2Hash,
			},
			&b.StateDiff{
				BlockNumber: block2.Number().Int64(),
				BlockHash:   block2.Hash(),
				CreatedAccounts: map[common.Address]b.AccountDiff{
					account2Addr: {
						Nonce:        b.DiffUint64{Value: &nonce0},
						Balance:      b.DiffBigInt{Value: big.NewInt(balanceChange1000)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
					contractAddr: {
						Nonce:        b.DiffUint64{Value: &nonce1},
						Balance:      b.DiffBigInt{Value: big.NewInt(0)},
						CodeHash:     "0x753f98a8d4328b15636e46f66f2cb4bc860100aa17967cc145fcd17d1d4710ea",
						ContractRoot: b.DiffString{Value: &contractContractRoot},
						Storage: map[string]b.DiffStorage{
							originalStorageKey: {
								Key:   &originalStorageKey,
								Value: &originalStorageValue},
						},
					},
				},
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: map[common.Address]b.AccountDiff{
					testBankAddress: {
						Nonce:        b.DiffUint64{Value: &nonce2},
						Balance:      b.DiffBigInt{Value: big.NewInt(block1BankBalance - balanceChange1000)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
					account1Addr: {
						Nonce:        b.DiffUint64{Value: &nonce2},
						Balance:      b.DiffBigInt{Value: big.NewInt(block1Account1Balance - balanceChange1000 + balanceChange1000)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
					burnAddress: {
						Nonce:        b.DiffUint64{Value: &nonce0},
						Balance:      b.DiffBigInt{Value: big.NewInt(miningReward + miningReward)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			arguments{
				oldStateRoot: block2.Root(),
				newStateRoot: block3.Root(),
				blockNumber:  block3.Number().Int64(),
				blockHash:    block3.Hash(),
			},
			&b.StateDiff{
				BlockNumber:     block3.Number().Int64(),
				BlockHash:       block3.Hash(),
				CreatedAccounts: map[common.Address]b.AccountDiff{},
				DeletedAccounts: emptyAccountDiffEventualMap,
				UpdatedAccounts: map[common.Address]b.AccountDiff{
					account2Addr: {
						Nonce:        b.DiffUint64{Value: &nonce0},
						Balance:      b.DiffBigInt{Value: big.NewInt(block2Account2Balance + miningReward)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
					contractAddr: {
						Nonce:        b.DiffUint64{Value: &nonce1},
						Balance:      b.DiffBigInt{Value: big.NewInt(0)},
						CodeHash:     "0x753f98a8d4328b15636e46f66f2cb4bc860100aa17967cc145fcd17d1d4710ea",
						ContractRoot: b.DiffString{Value: &newContractRoot},
						Storage: map[string]b.DiffStorage{
							"0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace": {
								Key:   &updatedStorageKey,
								Value: &updatedStorageValue},
						},
					},
					testBankAddress: {
						Nonce:        b.DiffUint64{Value: &nonce3},
						Balance:      b.DiffBigInt{Value: big.NewInt(99989000)},
						CodeHash:     "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
						ContractRoot: b.DiffString{Value: &originalContractRoot},
						Storage:      map[string]b.DiffStorage{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		arguments := test.startingArguments
		diff, err := builder.BuildStateDiff(arguments.oldStateRoot, arguments.newStateRoot, arguments.blockNumber, arguments.blockHash)
		if err != nil {
			t.Error(err)
		}

		fields := []string{"BlockNumber", "BlockHash", "DeletedAccounts", "UpdatedAccounts", "CreatedAccounts"}

		for _, field := range fields {
			reflectionOfDiff := reflect.ValueOf(diff)
			diffValue := reflect.Indirect(reflectionOfDiff).FieldByName(field)

			reflectionOfExpected := reflect.ValueOf(test.expected)
			expectedValue := reflect.Indirect(reflectionOfExpected).FieldByName(field)

			diffValueInterface := diffValue.Interface()
			expectedValueInterface := expectedValue.Interface()

			if !equals(diffValueInterface, expectedValueInterface) {
				t.Logf("Test failed: %s", test.name)
				t.Errorf("field: %+v\nactual: %+v\nexpected: %+v", field, diffValueInterface, expectedValueInterface)
			}
		}
	}
}

func equals(actual, expected interface{}) (success bool) {
	if actualByteSlice, ok := actual.([]byte); ok {
		if expectedByteSlice, ok := expected.([]byte); ok {
			return bytes.Equal(actualByteSlice, expectedByteSlice)
		}
	}

	return reflect.DeepEqual(actual, expected)
}

// makeChain creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent. In addition, every 3rd block
// contains a transaction and every 5th an uncle to allow testing correct block
// reassembly.
func makeChain(n int, parent *types.Block) ([]common.Hash, map[common.Hash]*types.Block) {
	blocks, _ := core.GenerateChain(params.TestChainConfig, parent, ethash.NewFaker(), testdb, n, testChainGen)
	hashes := make([]common.Hash, n+1)
	hashes[len(hashes)-1] = parent.Hash()
	blockm := make(map[common.Hash]*types.Block, n+1)
	blockm[parent.Hash()] = parent
	for i, b := range blocks {
		hashes[len(hashes)-i-2] = b.Hash()
		blockm[b.Hash()] = b
	}
	return hashes, blockm
}

func testChainGen(i int, block *core.BlockGen) {
	signer := types.HomesteadSigner{}
	switch i {
	case 0:
		// In block 1, the test bank sends account #1 some ether.
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), account1Addr, big.NewInt(10000), params.TxGas, nil, nil), signer, testBankKey)
		block.AddTx(tx)
	case 1:
		// In block 2, the test bank sends some more ether to account #1.
		// account1Addr passes it on to account #2.
		// account1Addr creates a test contract.
		tx1, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), account1Addr, big.NewInt(1000), params.TxGas, nil, nil), signer, testBankKey)
		nonce := block.TxNonce(account1Addr)
		tx2, _ := types.SignTx(types.NewTransaction(nonce, account2Addr, big.NewInt(1000), params.TxGas, nil, nil), signer, account1Key)
		nonce++
		tx3, _ := types.SignTx(types.NewContractCreation(nonce, big.NewInt(0), 1000000, big.NewInt(0), contractCode), signer, account1Key)
		contractAddr = crypto.CreateAddress(account1Addr, nonce) //0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592
		block.AddTx(tx1)
		block.AddTx(tx2)
		block.AddTx(tx3)
	case 2:
		// Block 3 is empty but was mined by account #2.
		block.SetCoinbase(account2Addr)
		//get function: 60cd2685
		//put function: c16431b9
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003")
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), contractAddr, big.NewInt(0), 100000, nil, data), signer, testBankKey)
		block.AddTx(tx)
	}
}

/*
contract test {

    uint256[100] data;

	constructor() public {
		data = [1];
	}

    function Put(uint256 addr, uint256 value) {
        data[addr] = value;
    }

    function Get(uint256 addr) constant returns (uint256 value) {
        return data[addr];
    }
}
*/
