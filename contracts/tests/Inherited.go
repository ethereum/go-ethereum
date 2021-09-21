// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package tests

import (
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/contracts/tests/contract"
)

type MyInherited struct {
	*contract.InheritedSession
	contractBackend bind.ContractBackend
}

func NewMyInherited(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*MyInherited, error) {
	smartContract, err := contract.NewInherited(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &MyInherited{
		&contract.InheritedSession{
			Contract:     smartContract,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployMyInherited(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *MyInherited, error) {
	contractAddr, _, _, err := contract.DeployInherited(transactOpts, contractBackend)
	if err != nil {
		return contractAddr, nil, err
	}
	smartContract, err := NewMyInherited(transactOpts, contractAddr, contractBackend)
	if err != nil {
		return contractAddr, nil, err
	}

	return contractAddr, smartContract, nil
}
