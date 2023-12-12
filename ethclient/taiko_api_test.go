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

func TestGetL2ParentHashes(t *testing.T) {
	ec, blocks, _ := newTaikoAPITestClient(t)

	hashes, err := ec.GetL2ParentHashes(context.Background(), common.Big3.Uint64())
	require.Nil(t, err)

	log.Info("hashes", "a", hashes)
	require.Equal(t, 3, len(hashes))
	require.Equal(t, blocks[0].Hash(), hashes[0])
	require.Equal(t, blocks[1].Hash(), hashes[1])
	require.Equal(t, blocks[2].Hash(), hashes[2])

	hashes, err = ec.GetL2ParentHashes(context.Background(), common.Big2.Uint64())
	require.Nil(t, err)

	require.Equal(t, 2, len(hashes))
	require.Equal(t, blocks[0].Hash(), hashes[0])
	require.Equal(t, blocks[1].Hash(), hashes[1])

	hashes, err = ec.GetL2ParentHashes(context.Background(), common.Big0.Uint64())
	require.Nil(t, err)

	require.Equal(t, 0, len(hashes))
}

func TestGetL2ParentBlocks(t *testing.T) {
	ec, blocks, _ := newTaikoAPITestClient(t)

	res, err := ec.GetL2ParentHeaders(context.Background(), common.Big3.Uint64())
	require.Nil(t, err)

	require.Equal(t, 3, len(res))
	require.Equal(t, res[0]["hash"], blocks[0].Hash().String())
	require.Equal(t, res[1]["hash"], blocks[1].Hash().String())
	require.Equal(t, res[2]["hash"], blocks[2].Hash().String())

	res, err = ec.GetL2ParentHeaders(context.Background(), common.Big2.Uint64())
	require.Nil(t, err)

	require.Equal(t, 2, len(res))
	require.Equal(t, res[0]["hash"], blocks[0].Hash().String())
	require.Equal(t, res[1]["hash"], blocks[1].Hash().String())

	res, err = ec.GetL2ParentHeaders(context.Background(), common.Big0.Uint64())
	require.Nil(t, err)

	require.Equal(t, 0, len(res))
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
