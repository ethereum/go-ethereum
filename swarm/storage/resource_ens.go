package storage

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type ENSResourceHandler struct {
	ResourceHandler
	ethapi *ethclient.Client
	ensapi *ens.ENS
}

func NewENSResourceHandler(privKey *ecdsa.PrivateKey, datadir string, cloudStore CloudStore, rpcClient *rpc.Client, ensAddr common.Address) (*ENSResourceHandler, error) {
	transactOpts := bind.NewKeyedTransactor(privKey)
	ethapi := ethclient.NewClient(rpcClient)
	ensinstance, err := ens.NewENS(transactOpts, ensAddr, ethapi)
	if err != nil {
		return nil, err
	}
	rh, err := NewRawResourceHandler(privKey, datadir, cloudStore, rpcClient, ens.EnsNode)
	if err != nil {
		return nil, err
	}
	rh.nameHashFunc = func(name string) common.Hash {
		return ens.EnsNode(name)
	}

	return &ENSResourceHandler{
		ResourceHandler: rh,
		ethapi:          ethclient.NewClient(rpcClient),
		ensapi:          ensinstance,
	}, nil
}
