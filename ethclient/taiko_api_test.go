package ethclient

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var (
	testKey, _   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr     = crypto.PubkeyToAddress(testKey.PublicKey)
	testContract = common.HexToAddress("0xbeef")
	testEmpty    = common.HexToAddress("0xeeee")
	testSlot     = common.HexToHash("0xdeadbeef")
	testValue    = crypto.Keccak256Hash(testSlot[:])
	testBalance  = big.NewInt(2e15)
)

func generateTestChain() (*core.Genesis, []*types.Block) {
	genesis := &core.Genesis{
		Config: params.AllEthashProtocolChanges,
		Alloc: types.GenesisAlloc{
			testAddr:     {Balance: testBalance, Storage: map[common.Hash]common.Hash{testSlot: testValue}},
			testContract: {Nonce: 1, Code: []byte{0x13, 0x37}},
			testEmpty:    {Balance: big.NewInt(1)},
		},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 1, generate)
	blocks = append([]*types.Block{genesis.ToBlock()}, blocks...)
	return genesis, blocks
}

func newTaikoAPITestClient(t *testing.T) (*Client, []*types.Block, ethdb.Database) {
	// Generate test chain.
	genesis, blocks := generateTestChain()

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
