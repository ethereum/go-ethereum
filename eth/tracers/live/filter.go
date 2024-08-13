package live

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func init() {
	tracers.LiveDirectory.Register("filter", newFilter)
}

const (
	tableSize = 2 * 1024 * 1024 * 1024
)

type traceResult struct {
	TxHash common.Hash `json:"txHash,omitempty"` // transaction hash
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

type filter struct {
	backend    tracers.Backend
	kvdb       ethdb.Database
	frdb       *rawdb.Freezer
	blockCh    chan uint64
	stopCh     chan struct{}
	tables     map[string]bool
	traces     map[string][]*traceResult
	tracer     *native.MuxTracer
	latest     atomic.Uint64
	offset     atomic.Uint64
	finalized  atomic.Uint64
	hash       common.Hash
	once       sync.Once
	offsetFile string
}

type filterTracerConfig struct {
	Path   string          `json:"path"` // Path to the directory where the tracer logs will be stored
	Config json.RawMessage `json:"config"`
}

func toTraceTable(name string) string {
	return name + "_tracers"
}

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func toKVKey(name string, number uint64, hash common.Hash) []byte {
	var typo byte
	switch name {
	case "callTracer":
		typo = byte('C')
	case "flatCallTracer":
		typo = byte('P')
	default:
		panic("not supported yet")
	}
	// TODO: have some prefix?
	key := append(encodeBlockNumber(number), hash.Bytes()...)
	key = append(key, typo)

	return key
}

func newFilter(cfg json.RawMessage, backend tracers.Backend) (*tracing.Hooks, []rpc.API, error) {
	var config filterTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, nil, fmt.Errorf("failed to parse config: %v", err)
		}
	}
	if config.Path == "" {
		return nil, nil, errors.New("filter tracer output path is required")
	}

	t, err := native.NewMuxTracer(config.Config)
	if err != nil {
		return nil, nil, err
	}

	var (
		kvpath = path.Join(config.Path, "trace")
		frpath = path.Join(config.Path, "freeze")
	)

	kvdb, err := rawdb.NewPebbleDBDatabase(kvpath, 128, 1024, "trace", false, false)
	if err != nil {
		return nil, nil, err
	}

	muxTracers := t.Tracers()
	tables := make(map[string]bool, len(muxTracers))
	traces := make(map[string][]*traceResult, len(muxTracers))
	for name := range muxTracers {
		tables[toTraceTable(name)] = false
		traces[name] = nil
	}

	frdb, err := rawdb.NewFreezer(frpath, "trace", false, tableSize, tables)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace freezer db: %v", err)
	}

	tail, err := frdb.Tail()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the tail block number from the freezer db: %v", err)
	}
	frozen, err := frdb.Ancients()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the frozen block numbers from the freezer db: %v", err)
	}

	f := &filter{
		backend:    backend,
		kvdb:       kvdb,
		frdb:       frdb,
		blockCh:    make(chan uint64, 100),
		stopCh:     make(chan struct{}),
		tables:     tables,
		traces:     traces,
		tracer:     t,
		offsetFile: path.Join(frpath, "OFFSET"),
	}
	offset := 0
	if _, err := os.Stat(f.offsetFile); err == nil || os.IsExist(err) {
		data, err := os.ReadFile(f.offsetFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read the offset from the freezer db: %v", err)
		}
		offset, err = strconv.Atoi(string(data))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert offset: %v", err)
		}
	}
	log.Info("Open filter tracer", "path", config.Path, "offset", offset, "tables", tables)

	f.latest.Store(tail + frozen + uint64(offset))
	f.offset.Store(uint64(offset))
	hooks := &tracing.Hooks{
		OnBlockStart: f.OnBlockStart,
		OnBlockEnd:   f.OnBlockEnd,
		OnTxStart:    f.OnTxStart,
		OnTxEnd:      f.OnTxEnd,

		// reuse the mux's hooks
		OnEnter:         t.OnEnter,
		OnExit:          t.OnExit,
		OnOpcode:        t.OnOpcode,
		OnFault:         t.OnFault,
		OnGasChange:     t.OnGasChange,
		OnBalanceChange: t.OnBalanceChange,
		OnNonceChange:   t.OnNonceChange,
		OnCodeChange:    t.OnCodeChange,
		OnStorageChange: t.OnStorageChange,
		OnLog:           t.OnLog,
	}
	apis := []rpc.API{
		{
			Namespace: "trace",
			Service:   &filterAPI{backend: backend, filter: f},
		},
	}

	// Initialize head if it doesn't exist
	head, _ := f.getFreezerHeadTail()
	if head == 0 {
		f.updateFreezerHead(f.latest.Load())
	}
	go f.freeze()

	return hooks, apis, nil
}

func (f *filter) OnBlockStart(ev tracing.BlockEvent) {
	// track the latest block number
	blknum := ev.Block.NumberU64()
	f.latest.Store(blknum)
	f.hash = ev.Block.Hash()
	if ev.Finalized != nil {
		f.finalized.Store(ev.Finalized.Number.Uint64())
	}

	// reset local cache
	txs := ev.Block.Transactions().Len()
	for name := range f.traces {
		f.traces[name] = make([]*traceResult, 0, txs)
	}

	// save the earliest arrived blknum as the offset
	f.once.Do(func() {
		if _, err := os.Stat(f.offsetFile); err != nil && os.IsNotExist(err) {
			f.offset.Store(blknum)
			os.WriteFile(f.offsetFile, []byte(fmt.Sprintf("%d", blknum)), 0666)
		}
	})
}

func (f *filter) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	f.tracer.OnTxStart(env, tx, from)
}

func (f *filter) OnTxEnd(receipt *types.Receipt, err error) {
	f.tracer.OnTxEnd(receipt, err)

	for name, tt := range f.tracer.Tracers() {
		trace := &traceResult{TxHash: receipt.TxHash}
		result, err := tt.GetResult()
		if err != nil {
			log.Error("Failed to get tracer results", "number", f.latest.Load(), "error", err)
			trace.Error = err.Error()
		} else {
			trace.Result = result
		}
		f.traces[name] = append(f.traces[name], trace)
	}
}

func (f *filter) OnBlockEnd(err error) {
	if err != nil {
		log.Warn("OnBlockEnd", "latest", f.latest.Load(), "error", err)
	}
	batch := f.kvdb.NewBatch()

	number := f.latest.Load()
	hash := f.hash
	for name, traces := range f.traces {
		data, err := json.Marshal(traces)
		if err != nil {
			log.Error("Failed to marshal traces", "error", err)
			break
		}
		batch.Put(toKVKey(name, number, hash), data)
	}
	if err := batch.Write(); err != nil {
		log.Error("Failed to write", "error", err)
		return
	}

	select {
	case f.blockCh <- f.finalized.Load():
	default:
		// Channel is full, log a warning
		log.Warn("Block channel is full, skipping finalized block notification")
	}
}

func (f *filter) readBlockTraces(ctx context.Context, name string, blknum uint64) ([]*traceResult, error) {
	if blknum > f.latest.Load() {
		return nil, errors.New("notfound")
	}
	if blknum < f.offset.Load() {
		return nil, errors.New("historical data not available")
	}

	_, tail := f.getFreezerHeadTail()

	// If tail is 0 (not found in kvdb), use the offset
	if tail == 0 {
		tail = f.offset.Load()
	}

	// Determine whether to read from kvdb or frdb
	var (
		data []byte
		err  error
	)
	if blknum >= tail {
		// Data is in kvdb
		data, err = f.readFromKVDB(ctx, name, blknum)
	} else {
		// Data is in frdb
		data, err = f.readFromFRDB(name, blknum)
	}
	if err != nil {
		return nil, err
	}

	var traces []*traceResult
	err = json.Unmarshal(data, &traces)
	return traces, err
}

func (f *filter) readFromKVDB(ctx context.Context, name string, blknum uint64) ([]byte, error) {
	header, err := f.backend.HeaderByNumber(ctx, rpc.BlockNumber(blknum))
	if err != nil {
		return nil, err
	}

	kvKey := toKVKey(name, blknum, header.Hash())
	data, err := f.kvdb.Get(kvKey)
	if err != nil {
		return nil, fmt.Errorf("traces not found in kvdb for block %d: %w", blknum, err)
	}
	return data, err
}

func (f *filter) readFromFRDB(name string, blknum uint64) ([]byte, error) {
	table := toTraceTable(name)
	var data []byte
	err := f.frdb.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		var err error
		data, err = reader.Ancient(table, blknum-f.offset.Load())
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("traces not found in frdb for block %d: %w", blknum, err)
	}
	return data, nil
}

func (f *filter) Close() {
	close(f.stopCh)

	if err := f.kvdb.Close(); err != nil {
		log.Error("Close kvdb failed", "err", err)
	}

	if err := f.frdb.Close(); err != nil {
		log.Error("Close freeze db failed", "err", err)
	}
}
