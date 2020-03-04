package bor_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/consensus/bor"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/core/vm"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth"
	"github.com/maticnetwork/bor/internal/ethapi"
	// "github.com/maticnetwork/bor/node"
	// "github.com/maticnetwork/bor/params"
)

func TestIsValidatorAction(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		// signer = new(types.HomesteadSigner)
	)
	data, err := ioutil.ReadFile("genesis.json")
	if err != nil {
		t.Fatalf("%s", err)
	}
	config := eth.Config
		Genesis: &core.Genesis{},
	}
	// var genesis core.Genesis
	if err := json.Unmarshal(data, config.Genesis); err != nil {
		t.Fatalf("%s", err)
	}
	// fmt.Println(genesis)
	chainConfig, _, err := core.SetupGenesisBlock(db, config.Genesis)
	fmt.Printf("Chain config: %v\n", chainConfig)

	// copy(genspec.ExtraData[extraVanity:], addr[:])
	config.Genesis.MustCommit(db)

	_eth := &eth.Ethereum{
		// config:         config,
		// chainDb:        chainDb,
		// eventMux:       ctx.EventMux,
		// accountManager: ctx.AccountManager,
		// engine:         nil,
		// shutdownChan:   make(chan bool),
		// networkID:      config.NetworkId,
		// gasPrice:       config.Miner.GasPrice,
		// etherbase:      config.Miner.Etherbase,
		// bloomRequests:  make(chan chan *bloombits.Retrieval),
		// bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
	}

	_eth.APIBackend = &eth.EthAPIBackend{false, _eth, nil}
	ethAPI := ethapi.NewPublicBlockChainAPI(_eth.APIBackend)
	// ctx := &node.ServiceContext{
	// 	config: &node.Config{},
	// 	// services:       make(map[reflect.Type]Service),
	// 	// EventMux:       n.eventmux,
	// 	// AccountManager: n.accman,
	// }
	engine := bor.New(chainConfig, db, &ethapi.PublicBlockChainAPI{})
	chain, err := core.NewBlockChain(db, nil, chainConfig, engine, vm.Config{}, nil)
	if err != nil {
		t.Fatalf("%s", err)
	}

	tx := types.NewTransaction(
		0,
		addr, // to - Just a place holder
		big.NewInt(0), 0 /* fix gas limit */, big.NewInt(0),
		nil, // data
	)
	engine.IsValidatorAction(chain, addr, tx)
}
