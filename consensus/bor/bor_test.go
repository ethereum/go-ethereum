package bor

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil" //nolint:typecheck
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

func TestGenesisContractChange(t *testing.T) {
	t.Parallel()

	addr0 := common.Address{0x1}

	b := &Bor{
		config: &params.BorConfig{
			Sprint: map[string]uint64{
				"0": 10,
			}, // skip sprint transactions in sprint
			BlockAlloc: map[string]interface{}{
				// write as interface since that is how it is decoded in genesis
				"2": map[string]interface{}{
					addr0.Hex(): map[string]interface{}{
						"code":    hexutil.Bytes{0x1, 0x2},
						"balance": "0",
					},
				},
				"4": map[string]interface{}{
					addr0.Hex(): map[string]interface{}{
						"code":    hexutil.Bytes{0x1, 0x3},
						"balance": "0x1000",
					},
				},
				"6": map[string]interface{}{
					addr0.Hex(): map[string]interface{}{
						"code":    hexutil.Bytes{0x1, 0x4},
						"balance": "0x2000",
					},
				},
			},
		},
	}

	genspec := &core.Genesis{
		Alloc: map[common.Address]core.GenesisAccount{
			addr0: {
				Balance: big.NewInt(0),
				Code:    []byte{0x1, 0x1},
			},
		},
		Config: &params.ChainConfig{},
	}

	db := rawdb.NewMemoryDatabase()

	genesis := genspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))

	statedb, err := state.New(genesis.Root(), state.NewDatabase(db), nil)
	require.NoError(t, err)

	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, genspec, nil, b, vm.Config{}, nil, nil, nil)
	require.NoError(t, err)

	addBlock := func(root common.Hash, num int64) (common.Hash, *state.StateDB) {
		h := &types.Header{
			ParentHash: root,
			Number:     big.NewInt(num),
		}
		b.Finalize(chain, h, statedb, nil, nil, nil)

		// write state to database
		root, err := statedb.Commit(0, false)
		require.NoError(t, err)
		require.NoError(t, statedb.Database().TrieDB().Commit(root, true))

		statedb, err := state.New(h.Root, state.NewDatabase(db), nil)
		require.NoError(t, err)

		return root, statedb
	}

	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x1})

	root := genesis.Root()

	// code does not change, balance remains 0
	root, statedb = addBlock(root, 1)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x1})
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(0))

	// code changes 1st time, balance remains 0
	root, statedb = addBlock(root, 2)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x2})
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(0))

	// code same as 1st change, balance remains 0
	root, statedb = addBlock(root, 3)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x2})
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(0))

	// code changes 2nd time, balance updates to 4096
	root, statedb = addBlock(root, 4)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x3})
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(4096))

	// code same as 2nd change, balance remains 4096
	root, statedb = addBlock(root, 5)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x3})
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(4096))

	// code changes 3rd time, balance remains 4096
	_, statedb = addBlock(root, 6)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x4})
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(4096))
}

func TestEncodeSigHeaderJaipur(t *testing.T) {
	t.Parallel()

	// As part of the EIP-1559 fork in mumbai, an incorrect seal hash
	// was used for Bor that did not included the BaseFee. The Jaipur
	// block is a hard fork to fix that.
	h := &types.Header{
		Difficulty: new(big.Int),
		Number:     big.NewInt(1),
		Extra:      make([]byte, 32+65),
	}

	var (
		// hash for the block without the BaseFee
		hashWithoutBaseFee = common.HexToHash("0x1be13e83939b3c4701ee57a34e10c9290ce07b0e53af0fe90b812c6881826e36")
		// hash for the block with the baseFee
		hashWithBaseFee = common.HexToHash("0xc55b0cac99161f71bde1423a091426b1b5b4d7598e5981ad802cce712771965b")
	)

	// Jaipur NOT enabled and BaseFee not set
	hash := SealHash(h, &params.BorConfig{JaipurBlock: big.NewInt(10)})
	require.Equal(t, hash, hashWithoutBaseFee)

	// Jaipur enabled (Jaipur=0) and BaseFee not set
	hash = SealHash(h, &params.BorConfig{JaipurBlock: common.Big0})
	require.Equal(t, hash, hashWithoutBaseFee)

	h.BaseFee = big.NewInt(2)

	// Jaipur enabled (Jaipur=Header block) and BaseFee set
	hash = SealHash(h, &params.BorConfig{JaipurBlock: common.Big1})
	require.Equal(t, hash, hashWithBaseFee)

	// Jaipur NOT enabled and BaseFee set
	hash = SealHash(h, &params.BorConfig{JaipurBlock: big.NewInt(10)})
	require.Equal(t, hash, hashWithoutBaseFee)
}
