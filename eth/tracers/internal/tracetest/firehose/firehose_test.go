package firehose_test

import (
	"math/big"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestFirehoseChain(t *testing.T) {
	context := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(uint64(1)),
		Time:        1,
		Difficulty:  big.NewInt(2),
		GasLimit:    uint64(1000000),
		BaseFee:     big.NewInt(8),
	}

	tracer, tracingHooks, onClose := newFirehoseTestTracer(t)
	defer onClose()

	genesis, blockchain := newBlockchain(t, types.GenesisAlloc{}, context, tracingHooks)

	block := types.NewBlock(&types.Header{
		ParentHash:       genesis.ToBlock().Hash(),
		Number:           context.BlockNumber,
		Difficulty:       context.Difficulty,
		Coinbase:         context.Coinbase,
		Time:             context.Time,
		GasLimit:         context.GasLimit,
		BaseFee:          context.BaseFee,
		ParentBeaconRoot: ptr(common.Hash{}),
	}, nil, nil, trie.NewStackTrie(nil))

	blockchain.SetBlockValidatorAndProcessorForTesting(
		ignoreValidateStateValidator{core.NewBlockValidator(genesis.Config, blockchain)},
		core.NewStateProcessor(genesis.Config, blockchain.HeaderChain()),
	)

	n, err := blockchain.InsertChain(types.Blocks{block})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	genesisLine, blockLines, unknownLines := readTracerFirehoseLines(t, tracer)
	require.Len(t, unknownLines, 0, "Lines:\n%s", strings.Join(slicesMap(unknownLines, func(l unknownLine) string { return "- '" + string(l) + "'" }), "\n"))
	require.NotNil(t, genesisLine)
	blockLines.assertEquals(t, filepath.Join("testdata", t.Name()),
		firehoseBlockLineParams{"1", "8e6ee4b1054d94df1d8a51fb983447dc2e27a854590c3ac0061f994284be8150", "0", "845bad515694a416bab4b8d44e22cf97a8c894a8502110ab807883940e185ce0", "0", "1000000000"},
	)
}

func TestFirehosePrestate(t *testing.T) {
	testFolders := []string{
		"./testdata/TestFirehosePrestate/keccak256_too_few_memory_bytes_get_padded",
		"./testdata/TestFirehosePrestate/keccak256_wrong_diff",
		"./testdata/TestFirehosePrestate/suicide_double_withdraw",
		"./testdata/TestFirehosePrestate/extra_account_creations",
	}

	for _, folder := range testFolders {
		name := filepath.Base(folder)

		t.Run(name, func(t *testing.T) {
			tracer, tracingHooks, onClose := newFirehoseTestTracer(t)
			defer onClose()

			runPrestateBlock(t, filepath.Join(folder, "prestate.json"), tracingHooks)

			genesisLine, blockLines, unknownLines := readTracerFirehoseLines(t, tracer)
			require.Len(t, unknownLines, 0, "Lines:\n%s", strings.Join(slicesMap(unknownLines, func(l unknownLine) string { return "- '" + string(l) + "'" }), "\n"))
			require.NotNil(t, genesisLine)
			blockLines.assertOnlyBlockEquals(t, folder, 1)
		})
	}

}

func TestFirehose_EIP7702(t *testing.T) {
	// Copied from ./core/blockchain_test.go#L4180 (TestEIP7702)

	var (
		config  = *params.MergedTestChainConfig
		signer  = types.LatestSigner(&config)
		engine  = beacon.New(ethash.NewFaker())
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		aa      = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		bb      = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
		cc      = common.HexToAddress("0x000000000000000000000000000000000000cccc")
		funds   = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
	)

	gspec := &core.Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1: {Balance: funds},
			addr2: {Balance: funds},
			aa: { // The address 0xAAAA calls into addr2
				Code:    program.New().Call(nil, addr2, 1, 0, 0, 0, 0).Bytes(),
				Nonce:   0,
				Balance: big.NewInt(0),
			},
			bb: { // The address 0xBBBB sstores 42 into slot 42.
				Code:    program.New().Sstore(0x42, 0x42).Bytes(),
				Nonce:   0,
				Balance: big.NewInt(0),
			},
			cc: { // The address 0xCCCC sstores 42 into slot 42.
				Code:    program.New().Sstore(0x42, 0x42).Bytes(),
				Nonce:   0,
				Balance: big.NewInt(0),
			},
		},
	}

	// Sign authorization tuples.
	// The way the auths are combined, it becomes
	// 1. tx -> addr1 which is delegated to 0xaaaa
	// 2. addr1:0xaaaa calls into addr2:0xbbbb
	// 3. addr2:0xbbbb  writes to storage
	auth1, _ := types.SignSetCode(key1, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: aa,
		Nonce:   1,
	})
	auth2OverwrittenLaterInList, _ := types.SignSetCode(key2, types.SetCodeAuthorization{
		Address: cc,
		Nonce:   0,
	})
	auth3InvalidAuthority := auth2OverwrittenLaterInList
	auth3InvalidAuthority.V = 4
	auth4, _ := types.SignSetCode(key2, types.SetCodeAuthorization{
		Address: bb,
		Nonce:   1,
	})
	auth5InvalidNonce, _ := types.SignSetCode(key2, types.SetCodeAuthorization{
		Address: bb,
		Nonce:   1,
	})

	auth1Reset, _ := types.SignSetCode(key1, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
		Address: common.Address{},
		Nonce:   4,
	})

	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 3, func(i int, b *core.BlockGen) {
		b.SetCoinbase(aa)

		if i == 0 {
			txdata := &types.SetCodeTx{
				ChainID:   uint256.MustFromBig(gspec.Config.ChainID),
				Nonce:     0,
				To:        addr1,
				Gas:       500000,
				GasFeeCap: uint256.MustFromBig(newGwei(5)),
				GasTipCap: uint256.NewInt(2),
				AuthList:  []types.SetCodeAuthorization{auth1, auth2OverwrittenLaterInList, auth3InvalidAuthority, auth4, auth5InvalidNonce},
			}
			tx := types.MustSignNewTx(key1, signer, txdata)
			b.AddTx(tx)
		} else if i == 1 {
			txdata := &types.LegacyTx{
				Nonce:    2,
				To:       &addr1,
				Value:    big.NewInt(0),
				Gas:      500000,
				GasPrice: newGwei(500),
			}
			tx := types.MustSignNewTx(key1, signer, txdata)
			b.AddTx(tx)
		} else if i == 2 {
			txdata := &types.SetCodeTx{
				ChainID:   uint256.MustFromBig(gspec.Config.ChainID),
				Nonce:     3,
				To:        addr1,
				Gas:       500000,
				GasFeeCap: uint256.MustFromBig(newGwei(5)),
				GasTipCap: uint256.NewInt(2),
				AuthList:  []types.SetCodeAuthorization{auth1Reset},
			}
			tx := types.MustSignNewTx(key1, signer, txdata)
			b.AddTx(tx)
		}
	})

	tracer, tracingHooks, onClose := newFirehoseTestTracer(t)
	defer onClose()

	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, gspec, nil, engine, vm.Config{Tracer: tracingHooks}, nil)
	require.NoError(t, err, "failed to create tester chain")

	defer chain.Stop()
	n, err := chain.InsertChain(blocks)
	require.NoError(t, err, "failed to insert chain block %d", n)

	assertBlockEquals(t, tracer, filepath.Join("testdata", "TestEIP7702"), len(blocks))
}
