package api

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// swarm domain name registry and resolver
// the DNS instance can be directly wrapped in rpc.Api
type DNS struct {
	registrar registrar.VersionedRegistrar
}

func NewDNS(registrar registrar.VersionedRegistrar) *DNS {
	return &DNS{registrar}
}

// Register involves sending a transaction, sender is an account with funds
// the same account is used to register the authors of commits
func (self *DNS) Register(sender common.Address, domain string, hash common.Hash) (err error) {
	domainhash := common.BytesToHash(crypto.Sha3([]byte(domain)))

	if self.registrar != nil {
		glog.V(logger.Debug).Infof("[DNR]: host '%s' (hash: '%v') to be registered as '%v'", domain, domainhash.Hex(), hash.Hex())
		_, err = self.registrar.Registry().SetHashToHash(sender, domainhash, hash)
	} else {
		err = fmt.Errorf("no registry: %v", err)
	}
	return
}

type ErrResolve error

func (self *DNS) Resolve(hostPort string) (contentHash storage.Key, err error) {
	host := hostPort
	var hash common.Hash
	var version *big.Int
	parts := domainAndVersion.Split(host, 3)
	if len(parts) > 1 && parts[1] != "" {
		host = parts[0]
		version = common.Big(parts[1])
	}
	hostHash := crypto.Sha3Hash([]byte(host))
	hash, err = self.registrar.Resolver(version).HashToHash(hostHash)
	if err != nil {
		err = fmt.Errorf("unable to resolve '%s': %v", hostPort, err)
	}
	contentHash = storage.Key(hash.Bytes())
	glog.V(logger.Debug).Infof("[DNR] resolve host '%s' to contentHash: '%v'", hostPort, contentHash)
	return
}
