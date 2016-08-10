package ens

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	name   = "my name on ENS"
	hash   = crypto.Sha3Hash([]byte("my content"))
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

func TestENS(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAccount{addr, big.NewInt(1000000000)})
	transactOpts := bind.NewKeyedTransactor(key)
	// Workaround for bug estimating gas in the call to Register
	transactOpts.GasLimit = big.NewInt(1000000)

	ens, err := DeployENS(transactOpts, contractBackend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	contractBackend.Commit()

	_, err = ens.Register(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	contractBackend.Commit()

	_, err = ens.SetContentHash(name, hash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	contractBackend.Commit()

	vhost, err := ens.Resolve(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if vhost.Hex() != hash.Hex()[2:] {
		t.Fatalf("resolve error, expected %v, got %v", hash.Hex(), vhost)
	}
}
