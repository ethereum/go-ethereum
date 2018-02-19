package pss

import (
	"github.com/ethereum/go-ethereum/p2p"
)

type protoCtrl struct {
	C        chan bool
	protocol *Protocol
	run      func(*p2p.Peer, p2p.MsgReadWriter) error
}
