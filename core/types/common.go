package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/wire"
)

type BlockProcessor interface {
	Process(*Block) (*big.Int, state.Messages, error)
}

type Broadcaster interface {
	Broadcast(wire.MsgType, []interface{})
}
