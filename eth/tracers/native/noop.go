package native

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/tracers"
)

func init() {
	tracers.RegisterNativeTracer("noopTracer", NewNoopTracer)
}

type noopTracer struct{}

func NewNoopTracer() tracers.Tracer {
	return &noopTracer{}
}

func (t *noopTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

func (t *noopTracer) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {
}

func (t *noopTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

func (t *noopTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

func (t *noopTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *noopTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (t *noopTracer) GetResult() (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

func (t *noopTracer) Stop(err error) {
}
