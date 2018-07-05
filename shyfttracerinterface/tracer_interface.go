package shyfttracerinterface

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
)

type IShyftTracer interface {
	GetTracerToRun(hash common.Hash, stack *node.Node) (interface{}, error)
}
