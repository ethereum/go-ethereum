package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
)

type Backend interface {
	BlockProcessor() *BlockProcessor
	ChainManager() *ChainManager
	TxPool() *TxPool
	PeerCount() int
	IsListening() bool
	Peers() []*p2p.Peer
	BlockDb() common.Database
	StateDb() common.Database
	EventMux() *event.TypeMux
}
