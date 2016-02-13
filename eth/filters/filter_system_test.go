package filters

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
)

func TestCallbacks(t *testing.T) {
	var (
		mux            event.TypeMux
		fs             = NewFilterSystem(&mux)
		blockDone      = make(chan struct{})
		txDone         = make(chan struct{})
		logDone        = make(chan struct{})
		removedLogDone = make(chan struct{})
		pendingLogDone = make(chan struct{})
	)

	blockFilter := &Filter{
		BlockCallback: func(*types.Block, vm.Logs) {
			close(blockDone)
		},
	}
	txFilter := &Filter{
		TransactionCallback: func(*types.Transaction) {
			close(txDone)
		},
	}
	logFilter := &Filter{
		LogCallback: func(l *vm.Log, oob bool) {
			if !oob {
				close(logDone)
			}
		},
	}
	removedLogFilter := &Filter{
		LogCallback: func(l *vm.Log, oob bool) {
			if oob {
				close(removedLogDone)
			}
		},
	}
	pendingLogFilter := &Filter{
		LogCallback: func(*vm.Log, bool) {
			close(pendingLogDone)
		},
	}

	fs.Add(blockFilter, ChainFilter)
	fs.Add(txFilter, PendingTxFilter)
	fs.Add(logFilter, LogFilter)
	fs.Add(removedLogFilter, LogFilter)
	fs.Add(pendingLogFilter, PendingLogFilter)

	mux.Post(core.ChainEvent{})
	mux.Post(core.TxPreEvent{})
	mux.Post(vm.Logs{&vm.Log{}})
	mux.Post(core.RemovedLogsEvent{vm.Logs{&vm.Log{}}})
	mux.Post(core.PendingLogsEvent{vm.Logs{&vm.Log{}}})

	const dura = 5 * time.Second
	failTimer := time.NewTimer(dura)
	select {
	case <-blockDone:
	case <-failTimer.C:
		t.Error("block filter failed to trigger (timeout)")
	}

	failTimer.Reset(dura)
	select {
	case <-txDone:
	case <-failTimer.C:
		t.Error("transaction filter failed to trigger (timeout)")
	}

	failTimer.Reset(dura)
	select {
	case <-logDone:
	case <-failTimer.C:
		t.Error("log filter failed to trigger (timeout)")
	}

	failTimer.Reset(dura)
	select {
	case <-removedLogDone:
	case <-failTimer.C:
		t.Error("removed log filter failed to trigger (timeout)")
	}

	failTimer.Reset(dura)
	select {
	case <-pendingLogDone:
	case <-failTimer.C:
		t.Error("pending log filter failed to trigger (timout)")
	}
}
