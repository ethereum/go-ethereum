package shared

import "strings"

const (
	AdminApiName    = "admin"
	EthApiName      = "eth"
	DbApiName       = "db"
	DebugApiName    = "debug"
	MergedApiName   = "merged"
	MinerApiName    = "miner"
	NetApiName      = "net"
	ShhApiName      = "shh"
	TxPoolApiName   = "txpool"
	PersonalApiName = "personal"
	Web3ApiName     = "web3"

	JsonRpcVersion = "2.0"
)

var (
	// All API's
	AllApis = strings.Join([]string{
		AdminApiName, DbApiName, EthApiName, DebugApiName, MinerApiName, NetApiName,
		ShhApiName, TxPoolApiName, PersonalApiName, Web3ApiName,
	}, ",")
)

