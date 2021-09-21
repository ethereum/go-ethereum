package trc21issuer

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/trc21issuer/contract"
	"math/big"
)

type MyTRC21 struct {
	*contract.MyTRC21Session
	contractBackend bind.ContractBackend
}

func NewTRC21(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*MyTRC21, error) {
	smartContract, err := contract.NewMyTRC21(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &MyTRC21{
		&contract.MyTRC21Session{
			Contract:     smartContract,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployTRC21(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, name string, symbol string, decimals uint8, cap, fee *big.Int) (common.Address, *MyTRC21, error) {
	contractAddr, _, _, err := contract.DeployMyTRC21(transactOpts, contractBackend, name, symbol, decimals, cap, fee)
	if err != nil {
		return contractAddr, nil, err
	}
	smartContract, err := NewTRC21(transactOpts, contractAddr, contractBackend)
	if err != nil {
		return contractAddr, nil, err
	}

	return contractAddr, smartContract, nil
}
