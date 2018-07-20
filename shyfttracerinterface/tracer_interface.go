package shyfttracerinterface

import (
	"github.com/ShyftNetwork/go-empyrean/common"
)

type IShyftTracer interface {
	GetTracerToRun(hash common.Hash) (interface{}, error)
}

