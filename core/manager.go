package core

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
)

type EthManager interface {
	BlockProcessor() *BlockProcessor
	ChainManager() *ChainManager
	TxPool() *TxPool
	PeerCount() int
	IsMining() bool
	IsListening() bool
	Peers() []*p2p.Peer
	KeyManager() *crypto.KeyManager
	Db() ethutil.Database
	EventMux() *event.TypeMux
}
