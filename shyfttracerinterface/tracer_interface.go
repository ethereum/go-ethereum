package shyfttracerinterface

import (
	"github.com/ethereum/go-ethereum/common"
)

type IShyftTracer interface {
	GetTracerToRun(hash common.Hash) (interface{}, error)
}

