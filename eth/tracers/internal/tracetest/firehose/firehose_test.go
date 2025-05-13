package firehose_test

import (
	"math/big"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type tracingModel string

const (
	tracingModelFirehose2_3 tracingModel = "fh2.3"
	tracingModelFirehose3_0 tracingModel = "fh3.0"
)

var tracingModels = []tracingModel{tracingModelFirehose2_3, tracingModelFirehose3_0}

func TestFirehosePrestate(t *testing.T) {
	testFolders := []string{
		"./testdata/TestFirehosePrestate/keccak256_too_few_memory_bytes_get_padded",
		"./testdata/TestFirehosePrestate/keccak256_wrong_diff",
		"./testdata/TestFirehosePrestate/suicide_double_withdraw",
		"./testdata/TestFirehosePrestate/extra_account_creations",
	}

	for _, concurrent := range []bool{true, false} {
		for _, folder := range testFolders {
			name := filepath.Base(folder)

			for _, model := range tracingModels {
				t.Run(string(model)+"/"+name, func(t *testing.T) {
					tracer, tracingHooks, onClose := newFirehoseTestTracer(t, model, concurrent)
					defer onClose()

					runPrestateBlock(t, filepath.Join(folder, "prestate.json"), tracingHooks)

					tracer.CloseBlockPrintQueue()

					genesisLine, blockLines, unknownLines := readTracerFirehoseLines(t, tracer)
					require.Len(t, unknownLines, 0, "Lines:\n%s", strings.Join(slicesMap(unknownLines, func(l unknownLine) string { return "- '" + string(l) + "'" }), "\n"))
					require.NotNil(t, genesisLine)
					blockLines.assertOnlyBlockEquals(t, filepath.Join(folder, string(model)), 1)
				})
			}
		}
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

	testBlockTracesCorrectly(t, gspec, engine, blocks, "TestEIP7702")
}

func TestFirehose_SystemCalls(t *testing.T) {
	gspec := &core.Genesis{
		Config: params.MergedTestChainConfig,
	}

	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *core.BlockGen) {})

	testBlockTracesCorrectly(t, gspec, engine, blocks, "TestSystemCalls")
}

func testBlockTracesCorrectly(t *testing.T, genesisSpec *core.Genesis, engine consensus.Engine, blocks []*types.Block, goldenDir string) {
	t.Helper()

	for _, concurrent := range []bool{true, false} {
		for _, model := range tracingModels {
			t.Run(string(model), func(t *testing.T) {
				tracer, tracingHooks, onClose := newFirehoseTestTracer(t, model, concurrent)
				defer onClose()

				chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, genesisSpec, nil, engine, vm.Config{Tracer: tracingHooks}, nil)
				require.NoError(t, err, "failed to create tester chain")

				chain.SetBlockValidatorAndProcessorForTesting(
					ignoreValidateStateValidator{core.NewBlockValidator(genesisSpec.Config, chain)},
					core.NewStateProcessor(genesisSpec.Config, chain.HeaderChain()),
				)

				defer chain.Stop()
				n, err := chain.InsertChain(blocks)
				require.NoError(t, err, "failed to insert chain block %d", n)

				tracer.CloseBlockPrintQueue()

				assertBlockEquals(t, tracer, filepath.Join("testdata", goldenDir, string(model)), len(blocks))
			})
		}
	}
}
