package bor

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall" //nolint:typecheck
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestGenesisContractChange(t *testing.T) {
	t.Parallel()

	addr0 := common.Address{0x1}

	b := &Bor{
		config: &params.BorConfig{
			Sprint: 10, // skip sprint transactions in sprint
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
	}

	db := rawdb.NewMemoryDatabase()
	genesis := genspec.MustCommit(db)

	statedb, err := state.New(genesis.Root(), state.NewDatabase(db), nil)
	require.NoError(t, err)

	config := params.ChainConfig{}
	chain, err := core.NewBlockChain(db, nil, &config, b, vm.Config{}, nil, nil, nil)
	require.NoError(t, err)

	addBlock := func(root common.Hash, num int64) (common.Hash, *state.StateDB) {
		h := &types.Header{
			ParentHash: root,
			Number:     big.NewInt(num),
		}
		b.Finalize(chain, h, statedb, nil, nil)

		// write state to database
		root, err := statedb.Commit(false)
		require.NoError(t, err)
		require.NoError(t, statedb.Database().TrieDB().Commit(root, true, nil))

		statedb, err := state.New(h.Root, state.NewDatabase(db), nil)
		require.NoError(t, err)

		return root, statedb
	}

	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x1})

	root := genesis.Root()

	// code does not change
	root, statedb = addBlock(root, 1)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x1})

	// code changes 1st time
	root, statedb = addBlock(root, 2)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x2})

	// code same as 1st change
	root, statedb = addBlock(root, 3)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x2})

	// code changes 2nd time
	_, statedb = addBlock(root, 4)
	require.Equal(t, statedb.GetCode(addr0), []byte{0x1, 0x3})

	// make sure balance change DOES NOT take effect
	require.Equal(t, statedb.GetBalance(addr0), big.NewInt(0))
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
	hash := SealHash(h, &params.BorConfig{JaipurBlock: 10})
	require.Equal(t, hash, hashWithoutBaseFee)

	// Jaipur enabled (Jaipur=0) and BaseFee not set
	hash = SealHash(h, &params.BorConfig{JaipurBlock: 0})
	require.Equal(t, hash, hashWithoutBaseFee)

	h.BaseFee = big.NewInt(2)

	// Jaipur enabled (Jaipur=Header block) and BaseFee set
	hash = SealHash(h, &params.BorConfig{JaipurBlock: 1})
	require.Equal(t, hash, hashWithBaseFee)

	// Jaipur NOT enabled and BaseFee set
	hash = SealHash(h, &params.BorConfig{JaipurBlock: 10})
	require.Equal(t, hash, hashWithoutBaseFee)
}

// TestCheckpoint can be used for to fetch checkpoint
// count and checkpoint for debugging purpose.
// Also, this is kept only for local use.
func TestCheckpoint(t *testing.T) {
	t.Skip()
	t.Parallel()

	ctx := context.Background()

	// TODO: For testing, add heimdall url here
	h := heimdall.NewHeimdallClient("http://localhost:1317")

	count, err := h.FetchCheckpointCount(ctx)
	if err != nil {
		t.Error(err)
	}

	t.Log("Count:", count)

	checkpoint1, err := h.FetchCheckpoint(ctx, count)
	if err != nil {
		t.Error(err)
	}

	t.Log("Checkpoint1:", checkpoint1)

	checkpoint2, err := h.FetchCheckpoint(ctx, 10000)
	if err != nil {
		t.Error(err)
	}

	t.Log("Checkpoint2:", checkpoint2)

	checkpoint3, err := h.FetchCheckpoint(ctx, -1)
	if err != nil {
		t.Error(err)
	}

	t.Log("Checkpoint3:", checkpoint3)

	if checkpoint3.RootHash != checkpoint1.RootHash {
		t.Fatal("Invalid root hash")
	}
}
