package storage

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
)

// ENS validation of mutable resource owners
type ENSValidator struct {
	api *ens.ENS
}

func NewENSValidator(contractaddress common.Address, backend bind.ContractBackend, transactOpts *bind.TransactOpts) (*ENSValidator, error) {
	var err error
	validator := &ENSValidator{}
	validator.api, err = ens.NewENS(transactOpts, contractaddress, backend)
	if err != nil {
		return nil, err
	}
	return validator, nil
}

func (self *ENSValidator) isOwner(name string, address common.Address) (bool, error) {
	owneraddr, err := self.api.Owner(self.nameHash(name))
	if err != nil {
		return false, err
	}
	return owneraddr == address, nil
}

func (self *ENSValidator) nameHash(name string) common.Hash {
	return ens.EnsNode(name)
}

// Default fallthrough validation of mutable resource ownership
type GenericValidator struct {
	hashFunc func(string) common.Hash
}

func NewGenericValidator(hashFunc func(string) common.Hash) *GenericValidator {
	return &GenericValidator{
		hashFunc: hashFunc,
	}
}
func (self *GenericValidator) isOwner(name string, address common.Address) (bool, error) {
	return true, nil
}

func (self *GenericValidator) nameHash(name string) common.Hash {
	return self.hashFunc(name)
}
