package XDCx

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/XDCx/contract"
)

type XDCXListing struct {
	*contract.XDCXListingSession
	contractBackend bind.ContractBackend
}

func NewMyXDCXListing(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*XDCXListing, error) {
	smartContract, err := contract.NewXDCXListing(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &XDCXListing{
		&contract.XDCXListingSession{
			Contract:     smartContract,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployXDCXListing(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *XDCXListing, error) {
	contractAddr, _, _, err := contract.DeployXDCXListing(transactOpts, contractBackend)
	if err != nil {
		return contractAddr, nil, err
	}
	smartContract, err := NewMyXDCXListing(transactOpts, contractAddr, contractBackend)
	if err != nil {
		return contractAddr, nil, err
	}

	return contractAddr, smartContract, nil
}
