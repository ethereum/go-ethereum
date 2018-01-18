package storage

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
)

type baseValidator struct {
	signFunc SignFunc
}

func (b *baseValidator) sign(datahash common.Hash) (Signature, error) {
	if b.signFunc == nil {
		return emptySignature, fmt.Errorf("No signature function")
	}
	return b.signFunc(datahash)
}

// ENS validation of mutable resource owners
type ENSValidator struct {
	*baseValidator
	api        *ens.ENS
	hashlength int
}

func NewENSValidator(contractaddress common.Address, backend bind.ContractBackend, transactOpts *bind.TransactOpts, signFunc SignFunc) (*ENSValidator, error) {
	var err error
	validator := &ENSValidator{
		baseValidator: &baseValidator{
			signFunc: signFunc,
		},
	}
	validator.api, err = ens.NewENS(transactOpts, contractaddress, backend)
	if err != nil {
		return nil, err
	}
	validator.hashlength = len(ens.EnsNode(dbDirName).Bytes())
	return validator, nil
}

func (self *ENSValidator) checkAccess(name string, address common.Address) (bool, error) {
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
	*baseValidator
	hashFunc   func(string) common.Hash
	hashlength int
}

func NewGenericValidator(hashFunc func(string) common.Hash, signFunc SignFunc) *GenericValidator {
	return &GenericValidator{
		baseValidator: &baseValidator{
			signFunc: signFunc,
		},
		hashFunc:   hashFunc,
		hashlength: len(hashFunc(dbDirName).Bytes()),
	}

}

func (self *GenericValidator) checkAccess(name string, address common.Address) (bool, error) {
	return true, nil
}

func (self *GenericValidator) nameHash(name string) common.Hash {
	return self.hashFunc(name)
}
