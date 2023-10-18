package ethclient

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func newTaikoAPITestClient(t *testing.T) (*Client, []*types.Block, ethdb.Database) {
	// Generate test chain.
	blocks := generateTestChain()

	// Create node
	n, err := node.New(&node.Config{})

	require.Nil(t, err)

	// Create Ethereum Service
	config := &ethconfig.Config{Genesis: genesis}
	ethservice, err := eth.New(n, config)
	require.Nil(t, err)

	n.RegisterAPIs([]rpc.API{
		{
			Namespace: "taiko",
			Version:   params.VersionWithMeta,
			Service:   eth.NewTaikoAPIBackend(ethservice),
			Public:    true,
		},
	})

	// Start node
	require.Nil(t, n.Start())

	// Insert test blocks
	_, err = ethservice.BlockChain().InsertChain(blocks[1:])

	require.Nil(t, err)

	return NewClient(n.Attach()), blocks, ethservice.ChainDb()
}

func TestHeadL1Origin(t *testing.T) {
	ec, blocks, db := newTaikoAPITestClient(t)

	headerHash := blocks[len(blocks)-1].Hash()

	l1OriginFound, err := ec.HeadL1Origin(context.Background())

	require.Equal(t, ethereum.NotFound.Error(), err.Error())
	require.Nil(t, l1OriginFound)

	testL1Origin := &rawdb.L1Origin{
		BlockID:       randomBigInt(),
		L2BlockHash:   headerHash,
		L1BlockHeight: randomBigInt(),
		L1BlockHash:   randomHash(),
	}

	rawdb.WriteL1Origin(db, testL1Origin.BlockID, testL1Origin)
	rawdb.WriteHeadL1Origin(db, testL1Origin.BlockID)

	l1OriginFound, err = ec.HeadL1Origin(context.Background())

	require.Nil(t, err)
	require.Equal(t, testL1Origin, l1OriginFound)
}

func TestL1OriginByID(t *testing.T) {
	ec, blocks, db := newTaikoAPITestClient(t)

	headerHash := blocks[len(blocks)-1].Hash()
	testL1Origin := &rawdb.L1Origin{
		BlockID:       randomBigInt(),
		L2BlockHash:   headerHash,
		L1BlockHeight: randomBigInt(),
		L1BlockHash:   randomHash(),
	}

	l1OriginFound, err := ec.L1OriginByID(context.Background(), testL1Origin.BlockID)
	require.Equal(t, ethereum.NotFound.Error(), err.Error())
	require.Nil(t, l1OriginFound)

	rawdb.WriteL1Origin(db, testL1Origin.BlockID, testL1Origin)
	rawdb.WriteHeadL1Origin(db, testL1Origin.BlockID)

	l1OriginFound, err = ec.L1OriginByID(context.Background(), testL1Origin.BlockID)

	require.Nil(t, err)
	require.Equal(t, testL1Origin, l1OriginFound)
}

// randomHash generates a random blob of data and returns it as a hash.
func randomHash() common.Hash {
	var hash common.Hash
	if n, err := rand.Read(hash[:]); n != common.HashLength || err != nil {
		panic(err)
	}
	return hash
}

// randomBigInt generates a random big integer.
func randomBigInt() *big.Int {
	randomBigInt, err := rand.Int(rand.Reader, common.Big256)
	if err != nil {
		log.Crit(err.Error())
	}

	return randomBigInt
}
