package storage

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
)

// ENS validation of mutable resource owners
type ENSValidator struct {
	owner common.Address
	api   *ens.ENS
}

func NewENSValidator(owneraddress common.Address, contractaddress common.Address, backend bind.ContractBackend, transactOpts *bind.TransactOpts) (*ENSValidator, error) {
	var err error
	validator := &ENSValidator{}
	validator.api, err = ens.NewENS(transactOpts, contractaddress, backend)
	if err != nil {
		return nil, err
	}
	validator.owner = owneraddress
	return validator, nil
}

func (self *ENSValidator) isOwner(name string) (bool, error) {
	owneraddr, err := self.api.Owner(self.nameHash(name))
	if err != nil {
		return false, err
	}
	return owneraddr == self.owner, nil
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
func (self *GenericValidator) isOwner(name string) (bool, error) {
	return true, nil
}

func (self *GenericValidator) nameHash(name string) common.Hash {
	return self.hashFunc(name)
}
