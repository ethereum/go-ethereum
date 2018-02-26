package storage

import (
	"errors"

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

type OwnerValidator interface {
	ValidateOwner(name string, address common.Address) (bool, error)
}

// ENS validation of mutable resource owners
type ENSValidator struct {
	*baseValidator
	api OwnerValidator
}

func NewENSValidator(contractaddress common.Address, ownerValidator OwnerValidator, signFunc SignFunc) *ENSValidator {
	return &ENSValidator{
		baseValidator: &baseValidator{
			signFunc: signFunc,
			hashsize: common.HashLength,
		},
		api: ownerValidator,
	}
}

func (self *ENSValidator) checkAccess(name string, address common.Address) (bool, error) {
	return self.api.ValidateOwner(name, address)
}

func (self *ENSValidator) NameHash(name string) common.Hash {
	return ens.EnsNode(name)
}
