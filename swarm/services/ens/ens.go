//go:generate abigen --sol contract/ENS.sol --pkg contract --out contract/ens.go

package ens

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/services/ens/contract"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// swarm domain name registry and resolver
type ENS struct {
	*contract.ENSSession
	contractBackend bind.ContractBackend
}

// NewENS creates a struct exposing convenient high-level operations for interacting with
// the Ethereum Name Service.
func NewENS(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*ENS, error) {
	ens, err := contract.NewENS(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &ENS{
		&contract.ENSSession{
			Contract:     ens,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

// DeployENS deploys an instance of the ENS nameservice, with a 'first in first served' root registrar.
func DeployENS(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend) (*ENS, error) {
	// Deploy the ENS registry
	ensAddr, _, _, err := contract.DeployENS(transactOpts, contractBackend, transactOpts.From)
	if err != nil {
		return nil, err
	}

	ens, err := NewENS(transactOpts, ensAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	// Deploy the registrar
	regAddr, _, _, err := contract.DeployFIFSRegistrar(transactOpts, contractBackend, ensAddr, [32]byte{})
	if err != nil {
		return nil, err
	}

	// Set the registrar as owner of the ENS root
	_, err = ens.SetOwner([32]byte{}, regAddr)
	if err != nil {
		return nil, err
	}

	return ens, nil
}

func ensParentNode(name string) (common.Hash, common.Hash) {
	parts := strings.SplitN(name, ".", 2)
	label := crypto.Keccak256Hash([]byte(parts[0]))
	if len(parts) == 1 {
		return [32]byte{}, label
	} else {
		parentNode, parentLabel := ensParentNode(parts[1])
		return crypto.Keccak256Hash(parentNode[:], parentLabel[:]), label
	}
}

func ensNode(name string) common.Hash {
	parentNode, parentLabel := ensParentNode(name)
	return crypto.Keccak256Hash(parentNode[:], parentLabel[:])
}

func (self *ENS) getResolver(node [32]byte) (*contract.PublicResolverSession, error) {
	resolverAddr, err := self.Resolver(node)
	if err != nil {
		return nil, err
	}

	resolver, err := contract.NewPublicResolver(resolverAddr, self.contractBackend)
	if err != nil {
		return nil, err
	}

	return &contract.PublicResolverSession{
		Contract:     resolver,
		TransactOpts: self.TransactOpts,
	}, nil
}

func (self *ENS) getRegistrar(node [32]byte) (*contract.FIFSRegistrarSession, error) {
	registrarAddr, err := self.Owner(node)
	if err != nil {
		return nil, err
	}

	registrar, err := contract.NewFIFSRegistrar(registrarAddr, self.contractBackend)
	if err != nil {
		return nil, err
	}

	return &contract.FIFSRegistrarSession{
		Contract:     registrar,
		TransactOpts: self.TransactOpts,
	}, nil
}

// Resolve is a non-transactional call that returns the content hash associated with a name.
func (self *ENS) Resolve(name string) (storage.Key, error) {
	node := ensNode(name)

	resolver, err := self.getResolver(node)
	if err != nil {
		return nil, err
	}

	ret, err := resolver.Content(node)
	if err != nil {
		return nil, err
	}

	return storage.Key(ret[:]), nil
}

// Register registers a new domain name for the caller, making them the owner of the new name.
// Only works if the registrar for the parent domain implements the FIFS registrar protocol.
func (self *ENS) Register(name string) (*types.Transaction, error) {
	parentNode, label := ensParentNode(name)

	registrar, err := self.getRegistrar(parentNode)
	if err != nil {
		return nil, err
	}

	opts := self.TransactOpts
	opts.GasLimit = big.NewInt(200000)
	return registrar.Contract.Register(&opts, label, self.TransactOpts.From)
}

// SetContentHash sets the content hash associated with a name. Only works if the caller
// owns the name, and the associated resolver implements a `setContent` function.
func (self *ENS) SetContentHash(name string, hash common.Hash) (*types.Transaction, error) {
	node := ensNode(name)

	resolver, err := self.getResolver(node)
	if err != nil {
		return nil, err
	}

	opts := self.TransactOpts
	opts.GasLimit = big.NewInt(200000)
	return resolver.Contract.SetContent(&opts, node, hash)
}
