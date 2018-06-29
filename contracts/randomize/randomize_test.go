package randomize

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
	addr   = crypto.PubkeyToAddress(key.PublicKey)
	byte0  = make([][32]byte, 2)
)

func TestRandomize(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}})
	transactOpts := bind.NewKeyedTransactor(key)

	_, randomize, err := DeployRandomize(transactOpts, contractBackend)
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}
	contractBackend.Commit()

	s, err := randomize.SetSecret(byte0)
	if err != nil {
		t.Fatalf("can't get secret: %v", err)
	}
	t.Log("secret", s)
	contractBackend.Commit()
}
