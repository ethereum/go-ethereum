package natspec

import (
	"github.com/ethereum/go-ethereum/xeth"
)

type StateReg struct {
	xeth             *xeth.XEth
	caURL, caNatSpec string //contract addresses
}

func NewStateReg(_xeth *xeth.XEth) (self *StateReg) {

	self.xeth = _xeth
	self.testCreateContracts()
	return

}

const codeURLhint = "0x33600081905550609c8060136000396000f30060003560e060020a900480632f926732" +
	"14601f578063f39ec1f714603157005b602b6004356024356044565b60006000f35b603a" +
	"600435607f565b8060005260206000f35b600054600160a060020a031633600160a06002" +
	"0a031614606257607b565b8060016000848152602001908152602001600020819055505b" +
	"5050565b60006001600083815260200190815260200160002054905091905056"

const codeNatSpec = "0x33600081905550609c8060136000396000f30060003560e060020a900480632f926732" +
	"14601f578063f39ec1f714603157005b602b6004356024356044565b60006000f35b603a" +
	"600435607f565b8060005260206000f35b600054600160a060020a031633600160a06002" +
	"0a031614606257607b565b8060016000848152602001908152602001600020819055505b" +
	"5050565b60006001600083815260200190815260200160002054905091905056"

func (self *StateReg) testCreateContracts() {

	var err error
	self.caURL, err = self.xeth.Transact(self.xeth.Coinbase(), "", "100000", "", self.xeth.DefaultGas().String(), codeURLhint)
	if err != nil {
		panic(err)
	}
	self.caNatSpec, err = self.xeth.Transact(self.xeth.Coinbase(), "", "100000", "", self.xeth.DefaultGas().String(), codeNatSpec)
	if err != nil {
		panic(err)
	}

}
