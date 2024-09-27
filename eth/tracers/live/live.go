package live

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

func init() {
	tracers.LiveDirectory.Register("live", newLive)
}

const (
	tableSize = 2 * 1024 * 1024 * 1024
)

type traceResult struct {
	TxHash *common.Hash `json:"txHash,omitempty"` // Transaction hash generated from block
	Result interface{}  `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string       `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// EncodeRLP implments rlp.Encoder
func (tr *traceResult) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{tr.Result, tr.Error})
}

// DecodeRLP implements rlp.Decoder
func (tr *traceResult) DecodeRLP(s *rlp.Stream) error {
	var temp struct {
		Result []byte
		Error  string
	}
	if err := s.Decode(&temp); err != nil {
		return err
	}
	tr.Error = temp.Error
	return json.Unmarshal(temp.Result, &tr.Result)
}

type live struct {
	backend    tracing.Backend
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

	enableNonceTracer bool
}

type liveTracerConfig struct {
	Path              string          `json:"path"` // Path to the directory where the tracer data will be stored
	Config            json.RawMessage `json:"config"`
	EnableNonceTracer bool            `json:"enableNonceTracer"`
}

func toTraceTable(name string) string {
	return name + "_tracers"
}

// encodeNumber encodes a number as big endian uint64
func encodeNumber(number uint64) []byte {
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
		typo = byte('F')
	case "prestateTracer":
		typo = byte('S')
	case "parityTracer":
		typo = byte('P')
	default:
		panic("not supported yet")
	}
	// TODO: have some prefix?
	key := append(encodeNumber(number), hash.Bytes()...)
	key = append(key, typo)

	return key
}

func newLive(cfg json.RawMessage, stack tracers.LiveApiRegister, backend tracing.Backend) (*tracing.Hooks, error) {
	var config liveTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %v", err)
		}
	}
	if config.Path == "" {
		return nil, errors.New("live tracer output path is required")
	}

	t, err := native.NewMuxTracer(config.Config)
	if err != nil {
		return nil, err
	}

	var (
		kvpath = path.Join(config.Path, "trace")
		frpath = path.Join(config.Path, "freeze")
	)

	kvdb, err := rawdb.NewPebbleDBDatabase(kvpath, 128, 1024, "trace", false, false)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to create trace freezer db: %v", err)
	}

	tail, err := frdb.Tail()
	if err != nil {
		return nil, fmt.Errorf("failed to read the tail block number from the freezer db: %v", err)
	}
	frozen, err := frdb.Ancients()
	if err != nil {
		return nil, fmt.Errorf("failed to read the frozen block numbers from the freezer db: %v", err)
	}

	l := &live{
		backend:    backend,
		kvdb:       kvdb,
		frdb:       frdb,
		blockCh:    make(chan uint64, 100),
		stopCh:     make(chan struct{}),
		tables:     tables,
		traces:     traces,
		tracer:     t,
		offsetFile: path.Join(frpath, "OFFSET"),

		enableNonceTracer: config.EnableNonceTracer,
	}
	offset := 0
	if _, err := os.Stat(l.offsetFile); err == nil || os.IsExist(err) {
		data, err := os.ReadFile(l.offsetFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read the offset from the freezer db: %v", err)
		}
		offset, err = strconv.Atoi(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to convert offset: %v", err)
		}
	}
	log.Info("Open live tracer", "path", config.Path, "offset", offset, "tables", tables)

	l.latest.Store(tail + frozen + uint64(offset))
	l.offset.Store(uint64(offset))
	hooks := &tracing.Hooks{
		OnBlockStart: l.OnBlockStart,
		OnBlockEnd:   l.OnBlockEnd,
		OnTxStart:    l.OnTxStart,
		OnTxEnd:      l.OnTxEnd,

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

	var apis []rpc.API
	if len(muxTracers) > 0 {
		apis = append(apis, rpc.API{Namespace: "trace", Service: &traceAPI{backend: backend, live: l}})
	}
	if config.EnableNonceTracer {
		apis = append(apis, rpc.API{Namespace: "eth", Service: &ethAPI{backend: backend, live: l}})
	}
	stack.RegisterAPIs(apis)

	go l.freeze()

	return hooks, nil
}

func (f *live) OnBlockStart(ev tracing.BlockEvent) {
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

func (f *live) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	if f.enableNonceTracer {
		key := append(from.Bytes(), encodeNumber(tx.Nonce())...)
		val := tx.Hash().Bytes()
		if err := f.kvdb.Put(key, val); err != nil {
			log.Warn("Failed to put nonce into kvdb", "err", err)
		}
	}
	f.tracer.OnTxStart(env, tx, from)
}

func (f *live) OnTxEnd(receipt *types.Receipt, err error) {
	f.tracer.OnTxEnd(receipt, err)

	for name, tt := range f.tracer.Tracers() {
		trace := &traceResult{}
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

func (f *live) OnBlockEnd(err error) {
	if err != nil {
		log.Warn("OnBlockEnd", "latest", f.latest.Load(), "error", err)
	}
	batch := f.kvdb.NewBatch()

	number := f.latest.Load()
	hash := f.hash
	for name, traces := range f.traces {
		data, err := rlp.EncodeToBytes(traces)
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

func (f *live) readBlockTraces(ctx context.Context, name string, blknum uint64) ([]*traceResult, error) {
	if blknum > f.latest.Load() {
		return nil, errors.New("notfound")
	}
	if blknum < f.offset.Load() {
		return nil, errors.New("historical data not available")
	}

	tail := f.getFreezerTail()

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
	err = rlp.DecodeBytes(data, &traces)
	return traces, err
}

func (f *live) readFromKVDB(ctx context.Context, name string, blknum uint64) ([]byte, error) {
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

func (f *live) readFromFRDB(name string, blknum uint64) ([]byte, error) {
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

func (f *live) Close() {
	close(f.stopCh)

	if err := f.kvdb.Close(); err != nil {
		log.Error("Close kvdb failed", "err", err)
	}

	if err := f.frdb.Close(); err != nil {
		log.Error("Close freeze db failed", "err", err)
	}
}
