package storage

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
)

type baseValidator struct {
	signFunc SignFunc
	hashsize int
}

func (b *baseValidator) sign(datahash common.Hash) (signature Signature, err error) {
	if b.signFunc == nil {
		return signature, errors.New("No signature function")
	}
	return b.signFunc(datahash)
}

func (b *baseValidator) hashSize() int {
	return b.hashsize
}

// ENS validation of mutable resource owners
type ENSValidator struct {
	*baseValidator
	api *ens.ENS
}

func NewENSValidator(contractaddress common.Address, backend bind.ContractBackend, transactOpts *bind.TransactOpts, signFunc SignFunc) (*ENSValidator, error) {
	var err error
	validator := &ENSValidator{
		baseValidator: &baseValidator{
			signFunc: signFunc,
			hashsize: common.HashLength,
		},
	}
	validator.api, err = ens.NewENS(transactOpts, contractaddress, backend)
	if err != nil {
		return nil, err
	}
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
