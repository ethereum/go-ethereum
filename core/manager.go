package core

import (
	"github.com/ethereum/go-ethereum/ethutil"
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
	Db() ethutil.Database
	EventMux() *event.TypeMux
}
