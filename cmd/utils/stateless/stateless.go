package stateless

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

// StatelessExecute executes the block contained in the Witness returning the post state root or an error
func StatelessExecute(chainCfg *params.ChainConfig, witness *state.Witness) (root common.Hash, err error) {
	rawDb := rawdb.NewMemoryDatabase()
	if err := witness.PopulateDB(rawDb); err != nil {
		return common.Hash{}, err
	}
	blob := rawdb.ReadAccountTrieNode(rawDb, nil)
	prestateRoot := crypto.Keccak256Hash(blob)

	db, err := state.New(prestateRoot, state.NewDatabaseWithConfig(rawDb, triedb.PathDefaults), nil)
	if err != nil {
		return common.Hash{}, err
	}
	engine := beacon.New(ethash.NewFaker())
	validator := core.NewBlockValidator(chainCfg, nil, engine)
	processor := core.NewStateProcessor(chainCfg, nil, engine)

	receipts, _, usedGas, err := processor.Process(witness.Block, db, vm.Config{}, witness)
	if err != nil {
		return common.Hash{}, err
	}

	// compute the state root.
	if root, err = validator.ValidateState(witness.Block, db, receipts, usedGas, false); err != nil {
		return common.Hash{}, err
	}
	return root, nil
}

// BuildStatelessProof executes a block, collecting the accessed pre-state into
// a Witness.  The RLP-encoded witness is returned.
func BuildStatelessProof(blockHash common.Hash, bc *core.BlockChain) ([]byte, error) {
	block := bc.GetBlockByHash(blockHash)
	if block == nil {
		return nil, fmt.Errorf("non-existent block %x", blockHash)
	} else if block.NumberU64() == 0 {
		return nil, fmt.Errorf("cannot build a stateless proof of the genesis block")
	}
	parentHash := block.ParentHash()
	parent := bc.GetBlockByHash(parentHash)
	if parent == nil {
		return nil, fmt.Errorf("block %x parent not present", parentHash)
	}

	db, err := bc.StateAt(parent.Header().Root)
	if err != nil {
		return nil, err
	}
	db.EnableWitnessBuilding()
	if bc.Snapshots() != nil {
		db.StartPrefetcher("BuildStatelessProof", false)
		defer db.StopPrefetcher()
	}
	stateProcessor := core.NewStateProcessor(bc.Config(), bc, bc.Engine())
	_, _, _, err = stateProcessor.Process(block, db, vm.Config{}, nil)
	if err != nil {
		return nil, err
	}
	if _, err = db.Commit(block.NumberU64(), true); err != nil {
		return nil, err
	}
	proof := db.Witness()
	proof.Block = block
	enc, err := proof.EncodeRLP()
	if err != nil {
		return nil, err
	}
	return enc, nil
}
