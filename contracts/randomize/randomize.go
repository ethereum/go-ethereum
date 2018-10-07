package randomize

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/randomize/contract"
	"math/big"
)

type Randomize struct {
	*contract.XDCRandomizeSession
	contractBackend bind.ContractBackend
}

func NewRandomize(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*Randomize, error) {
	randomize, err := contract.NewXDCRandomize(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &Randomize{
		&contract.XDCRandomizeSession{
			Contract:     randomize,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployRandomize(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *Randomize, error) {
	randomizeAddr, _, _, err := contract.DeployXDCRandomize(transactOpts, contractBackend, big.NewInt(2), big.NewInt(0), big.NewInt(1))
	if err != nil {
		return randomizeAddr, nil, err
	}

	randomize, err := NewRandomize(transactOpts, randomizeAddr, contractBackend)
	if err != nil {
		return randomizeAddr, nil, err
	}

	return randomizeAddr, randomize, nil
}