package bor

import (
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/rawdb"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/core/vm"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/internal/ethapi"
	"github.com/maticnetwork/bor/params"
)

type MockEthAPI struct{}

// func (ethapi *MockEthAPI) call()

func TestIsValidatorAction(t *testing.T) {
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		// bor = New(&params.ChainConfig{}, db, &ethapi.PublicBlockChainAPI{})
		// signer = new(types.HomesteadSigner)
	)
	genspec := &core.Genesis{
		ExtraData: make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc: map[common.Address]core.GenesisAccount{
			addr: {Balance: big.NewInt(1)},
		},
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])
	genspec.MustCommit(db)
	// genesis := genspec.MustCommit(db)
	config := &params.ChainConfig{
		Bor: &params.BorConfig{
			ValidatorContract:     "0x0000000000000000000000000000000000001000",
			StateReceiverContract: "0x0000000000000000000000000000000000001001",
		},
	}
	bor := New(config, db, &ethapi.PublicBlockChainAPI{})
	chain, err := core.NewBlockChain(db, nil, config, bor, vm.Config{}, nil)
	if err != nil {
		t.Fatalf("%s", err)
	}

	tx := types.NewTransaction(
		0,
		addr, // to - Just a place holder
		big.NewInt(0), 0 /* fix gas limit */, big.NewInt(0),
		nil, // data
	)
	bor.isValidatorAction(chain, addr, tx)
}
