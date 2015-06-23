package ethreg

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/xeth"
)

// implements a versioned Registrar on an archiving full node
type EthReg struct {
	backend  *xeth.XEth
	registry *registrar.Registrar
}

func New(xe *xeth.XEth) (self *EthReg) {
	self = &EthReg{backend: xe}
	self.registry = registrar.New(xe)
	return
}

func (self *EthReg) Registry() *registrar.Registrar {
	return self.registry
}

func (self *EthReg) Resolver(n *big.Int) *registrar.Registrar {
	xe := self.backend
	if n != nil {
		xe = self.backend.AtStateNum(n.Int64())
	}
	return registrar.New(xe)
}
