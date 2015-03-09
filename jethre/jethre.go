package javascript

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/xeth"
)

var jsrelogger = logger.NewLogger("JEthRE")

/*
JEthRE (jay - eth runtime environment) is a javascript environment that embeds ethereum.
It provides javascript bindings to the entire extended ethereum interface https://github.com/ethereum/go-ethereum/wiki/XEth

*Note* JEthRe is not to be confused with ethereum.js
JEthRE is more like an admin console for your ethereum client, not a helper lib for Dapps

The JEth javascript API is found on the wiki.
All exported functions in jeth map to a lowercase variant in the eth namespace within JEthRE

Note that the CLI allows you to execute a js script within JEthRE offering scriptable ethereum

*/

type JEthRE struct {
	*JSRE
	jeth *jeth
}

func NewJEthRE(ethereum *eth.Ethereum, assetPath string) (self *JEthRE) {
	re := NewJSRE(assetPath)
	self = &JEthRE{
		JSRE: re,
		jeth: &jeth{xeth.New(ethereum), re.toVal},
	}
	self.Load("bignumber.min.js")
	self.Bind("eth", self.jeth)
	jsrelogger.Infoln("Javascript-Ethereum Runtime Environment started")
	return
}
