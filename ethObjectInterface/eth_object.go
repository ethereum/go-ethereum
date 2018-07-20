package ethObjectInterface

import "github.com/ShyftNetwork/go-empyrean/eth"

//var ethObject *eth.Ethereum

func GetEthObject() (interface{}){
	return eth.EthObject
}