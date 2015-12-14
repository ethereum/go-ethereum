//go:generate abigen --sol contract/ens.sol	--pkg contract --out contract/ens.go
package ens

import (
	"fmt"
	"math/big"
	"regexp"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/services/ens/contract"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var domainAndVersion = regexp.MustCompile("[@:;,]+")

// swarm domain name registry and resolver
// the ENS instance can be directly wrapped in rpc.Api
type ENS struct {
	*contract.ENSSession
}

// NewENS creates a proxy instance wrapping the abigen interface to the ENS contract
// using the transaction options passed as first argument, it sets up a session
func NewENS(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) *ENS {
	ens, err := contract.NewENS(contractAddr, contractBackend)
	if err != nil {
		glog.V(logger.Debug).Infof("error setting up name server on %v, skipping: %v", contractAddr.Hex(), err)
	}
	return &ENS{
		&contract.ENSSession{
			Contract:     ens,
			TransactOpts: *transactOpts,
		},
	}
}

// Register(name, hash )
//involves sending a transaction (sent by sender specified as From of Transact)
func (self *ENS) Register(name string, hash common.Hash) (*types.Transaction, error) {
	namehash := crypto.Sha3Hash([]byte(name))
	owner, err := self.Owners(namehash)
	if err != nil {
		return nil, fmt.Errorf("error registering '%s': %v", name, err)
	}
	if (owner != common.Address{} && owner != self.TransactOpts.From) {
		return nil, fmt.Errorf("error registering '%s': already set as %", name)
	}
	glog.V(logger.Debug).Infof("[ENS]: host '%s' (hash: '%v') to be registered as '%v'", name, namehash.Hex(), hash.Hex())
	return self.Set(namehash, hash)
}

func (self *ENS) WhoseIs(name string) (common.Address, error) {
	namehash := crypto.Sha3Hash([]byte(name))
	return self.Owners(namehash)
}

// resolve is a non-tranasctional call, returns hash as storage.Key
func (self *ENS) Resolve(hostPort string) (storage.Key, error) {
	host := hostPort
	var version *big.Int
	parts := domainAndVersion.Split(host, 3)
	if len(parts) > 1 && parts[1] != "" {
		host = parts[0]
		version = common.Big(parts[1])
	}
	hash := crypto.Sha3Hash([]byte(host))
	_ = version
	// hash, err = self.registrar.Resolver(version).HashToHash(hostHash)
	hash, err := self.Registry(hash)
	if err != nil {
		return nil, fmt.Errorf("error resolving '%v': %v", hash.Hex(), err)
	}
	if (hash == common.Hash{}) {
		return nil, fmt.Errorf("unable to resolve '%v': not found", hash)
	}
	contentHash := storage.Key(hash.Bytes())
	glog.V(logger.Debug).Infof("[ENS] resolve host '%v' to contentHash: '%v'", hash, contentHash)
	return contentHash, nil
}
