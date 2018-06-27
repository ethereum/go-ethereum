package validator

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/validator/contract"
	"math/big"
)

type Validator struct {
	*contract.TomoValidatorSession
	contractBackend bind.ContractBackend
}

func NewValidator(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*Validator, error) {
	validator, err := contract.NewTomoValidator(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &Validator{
		&contract.TomoValidatorSession{
			Contract:     validator,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployValidator(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *Validator, error) {
	validatorAddr, _, _, err := contract.DeployTomoValidator(transactOpts, contractBackend, big.NewInt(50000), big.NewInt(99), big.NewInt(100))
	if err != nil {
		return validatorAddr, nil, err
	}

	validator, err := NewValidator(transactOpts, validatorAddr, contractBackend)
	if err != nil {
		return validatorAddr, nil, err
	}

	return validatorAddr, validator, nil
}
