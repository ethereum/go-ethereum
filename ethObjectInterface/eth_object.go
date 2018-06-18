package ethObjectInterface

import "github.com/ethereum/go-ethereum/eth"

//var ethObject *eth.Ethereum

func GetEthObject() (interface{}){
	return eth.EthObject
}