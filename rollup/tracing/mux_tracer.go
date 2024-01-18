package tracing

import (
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/vm"
	_ "github.com/scroll-tech/go-ethereum/eth/tracers/native"
)

// MuxTracer is a tracer mux, to support running multiple tracers together
type MuxTracer struct {
	tracers []vm.EVMLogger
}

// NewMuxTracer creates a new MuxTracer with multiple tracers
func NewMuxTracer(tracers ...vm.EVMLogger) *MuxTracer {
	return &MuxTracer{tracers}
}

// CaptureStart runs CaptureStart for each tracer in the MuxTracer
func (t *MuxTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	for _, tracer := range t.tracers {
		tracer.CaptureStart(env, from, to, create, input, gas, value)
	}
}

// CaptureState runs CaptureState for each tracer in the MuxTracer
func (t *MuxTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
	}
}

// CaptureStateAfter runs CaptureStateAfter for each tracer in the MuxTracer
func (t *MuxTracer) CaptureStateAfter(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureStateAfter(pc, op, gas, cost, scope, rData, depth, err)
	}
}

// CaptureEnter runs CaptureEnter for each tracer in the MuxTracer
func (t *MuxTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	for _, tracer := range t.tracers {
		tracer.CaptureEnter(typ, from, to, input, gas, value)
	}
}

// CaptureExit runs CaptureExit for each tracer in the MuxTracer
func (t *MuxTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureExit(output, gasUsed, err)
	}
}

// CaptureFault runs CaptureFault for each tracer in the MuxTracer
func (t *MuxTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
	}
}

// CaptureEnd runs CaptureEnd for each tracer in the MuxTracer
func (t *MuxTracer) CaptureEnd(output []byte, gasUsed uint64, d time.Duration, err error) {
	for _, tracer := range t.tracers {
		tracer.CaptureEnd(output, gasUsed, d, err)
	}
}
