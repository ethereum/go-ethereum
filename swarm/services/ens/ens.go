//go:generate abigen --sol contract/ens.sol	--pkg contract --out contract/ens.go
package ens

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/services/ens/contract"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var domainAndVersion = regexp.MustCompile("[@:;,]+")
var qtypeChash = [32]byte{ 0x43, 0x48, 0x41, 0x53, 0x48}
var rtypeChash = [16]byte{ 0x43, 0x48, 0x41, 0x53, 0x48}

// swarm domain name registry and resolver
// the ENS instance can be directly wrapped in rpc.Api
type ENS struct {
	transactOpts *bind.TransactOpts;
	contractBackend bind.ContractBackend;
	rootAddress common.Address;
}

func NewENS(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) *ENS {
	return &ENS{
		transactOpts: transactOpts,
		contractBackend: contractBackend,
		rootAddress: contractAddr,
	}
}

func (self *ENS) newResolver(contractAddr common.Address) (*contract.ResolverSession, error) {
	resolver, err := contract.NewResolver(contractAddr, self.contractBackend)
	if err != nil {
		return nil, err
	}
	return &contract.ResolverSession{
		Contract: resolver,
		TransactOpts: *self.transactOpts,
	}, nil
}

// resolve is a non-tranasctional call, returns hash as storage.Key
func (self *ENS) Resolve(hostPort string) (storage.Key, error) {
	host := hostPort
	parts := domainAndVersion.Split(host, 3)
	if len(parts) > 1 && parts[1] != "" {
		host = parts[0]
	}
	return self.resolveName(self.rootAddress, host)
}

func (self *ENS) nextResolver(resolver *contract.ResolverSession, nodeId [12]byte, label string) (*contract.ResolverSession, [12]byte, error) {
	hash := crypto.Sha3Hash([]byte(label))
	ret, err := resolver.FindResolver(nodeId, hash)
	if err != nil {
		err = fmt.Errorf("error resolving label '%v': %v", label, err)
		return nil, [12]byte{}, err
	}
	if ret.Rcode != 0 {
		err = fmt.Errorf("error resolving label '%v': got response code %v", label, ret.Rcode)
		return nil, [12]byte{}, err
	}
	nodeId = ret.Rnode;
	resolver, err = self.newResolver(ret.Raddress)
	if err != nil {
		return nil, [12]byte{}, err
	}

	return resolver, nodeId, nil
}

func (self *ENS) findResolver(rootAddress common.Address, host string) (*contract.ResolverSession, [12]byte, error) {
	resolver, err := self.newResolver(self.rootAddress)
	if err != nil {
		return nil, [12]byte{}, err
	}

	if len(host) == 0 {
		return resolver, [12]byte{}, nil
	}

	labels := strings.Split(host, ".")

	var nodeId [12]byte
	for i := len(labels) - 1; i >= 0; i-- {
		var err error
		resolver, nodeId, err = self.nextResolver(resolver, nodeId, labels[i])
		if err != nil {
			return nil, [12]byte{}, err
		}
	}

	return resolver, nodeId, nil
}

func (self *ENS) resolveName(rootAddress common.Address, host string) (storage.Key, error) {
	resolver, nodeId, err := self.findResolver(rootAddress, host)
	if err != nil {
		return nil, err
	}

	ret, err := resolver.Resolve(nodeId, qtypeChash, 0)
	if err != nil {
		return nil, fmt.Errorf("error looking up RR on '%v': %v", host, err)
	}
	if ret.Rcode != 0 {
		return nil, fmt.Errorf("error looking up RR on '%v': got response code %v", host, ret.Rcode)
	}
	return storage.Key(ret.Data[:]), nil
}

/**
 * Registers a new domain name for the caller, making them the owner of the new name.
 */
func (self *ENS) Register(name string, resolverAddress common.Address) (*types.Transaction, error) {
	// Find the resolver that we should register with (the one that controls the parent domain)
	parts := strings.SplitN(name, ".", 2)

	baseName := ""
	if len(parts) > 1 {
		baseName = parts[1]
	}

	resolver, nodeId, err := self.findResolver(self.rootAddress, baseName)
	if err != nil {
		return nil, err
	}
	if nodeId != [12]byte{} {
		return nil, fmt.Errorf("cannot register domains on %v: not a root node", baseName)
	}

	// Send it a register transaction
	hash := crypto.Sha3Hash([]byte(parts[0]))
	return resolver.Register(hash, resolverAddress, [12]byte{})
}

/**
 * Steps through name components until it finds a PersonalResolver contract.
 * Returns the resolver, the node ID, and the remaining name components.
 */
func (self *ENS) findPersonalResolver(name string) (*contract.ResolverSession, [12]byte, string, error) {
	var nodeId [12]byte

	resolver, err := self.newResolver(self.rootAddress)
	if err != nil {
		return nil, [12]byte{}, "", err
	}

	labels := strings.Split(name, ".")

	for i := len(labels) - 1; i >= 0; i-- {
		if personal, _ := resolver.IsPersonalResolver(); personal {
			return resolver, nodeId, strings.Join(labels[0:i + 1], "."), nil
		}

		resolver, nodeId, err = self.nextResolver(resolver, nodeId, labels[i])
		if err != nil {
			return nil, [12]byte{}, "", err
		}
	}

	if personal, _ := resolver.IsPersonalResolver(); !personal {
		return nil, [12]byte{}, "", fmt.Errorf("Personal resolver not found in any name component")
	} else {
		return resolver, nodeId, "", nil
	}
}

/**
 * Sets the content hash associated with a name.
 */
func (self *ENS) SetContentHash(name string, hash common.Hash) (*types.Transaction, error) {
	resolver, nodeId, name, err := self.findPersonalResolver(name)
	if err != nil {
		return nil, err
	}

	return resolver.SetRR(nodeId, name, rtypeChash, 3600, 20, [32]byte(hash))
}
