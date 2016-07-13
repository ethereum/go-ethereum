//go:generate abigen --abi contract/OpenRegistrar.abi --bin contract/OpenRegistrar.bin --pkg contract --type OpenRegistrar --out contract/OpenRegistrar.go
//go:generate abigen --abi contract/PersonalResolver.abi --bin contract/PersonalResolver.bin --pkg contract --type PersonalResolver --out contract/PersonalResolver.go
package ens

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/services/ens/contract"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	name   = "my name on ENS"
	hash   = crypto.Sha3Hash([]byte("my content"))
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

func deployRegistrar(prvKey *ecdsa.PrivateKey, amount *big.Int, backend *backends.SimulatedBackend) (common.Address, error) {
	deployTransactor := bind.NewKeyedTransactor(prvKey)
	deployTransactor.Value = amount
	addr, _, _, err := contract.DeployOpenRegistrar(deployTransactor, backend)
	if err != nil {
		return common.Address{}, err
	}
	backend.Commit()
	return addr, nil
}

func deployResolver(prvKey *ecdsa.PrivateKey, amount *big.Int, backend *backends.SimulatedBackend) (common.Address, error) {
	deployTransactor := bind.NewKeyedTransactor(prvKey)
	deployTransactor.Value = amount
	addr, _, _, err := contract.DeployPersonalResolver(deployTransactor, backend)
	if err != nil {
		return common.Address{}, err
	}
	backend.Commit()
	return addr, nil
}

func TestENS(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAccount{addr, big.NewInt(1000000000)})
	transactOpts := bind.NewKeyedTransactor(key)
	contractAddr, err := deployRegistrar(key, big.NewInt(0), contractBackend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resolverAddr, err := deployResolver(key, big.NewInt(0), contractBackend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ens := NewENS(transactOpts, contractAddr, contractBackend)
	_, err = ens.Register(name, resolverAddr)
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
