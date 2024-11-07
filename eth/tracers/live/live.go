package live

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
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
	backend   tracing.Backend
	kvdb      ethdb.Database
	frdb      *rawdb.Freezer
	freezeCh  chan uint64
	stopCh    chan struct{}
	tables    map[string]bool
	traces    map[string][]*traceResult
	tracer    *native.MuxTracer
	latest    atomic.Uint64
	offset    atomic.Uint64
	finalized uint64
	hash      common.Hash
	once      sync.Once

	enableNonceTracer bool
}

type liveTracerConfig struct {
	Path              string          `json:"path"` // Path to the directory where the tracer data will be stored
	Config            json.RawMessage `json:"config"`
	EnableNonceTracer bool            `json:"enableNonceTracer"`
	MaxKeepBlocks     uint64          `json:"maxKeepBlocks"` // Maximum number of blocks to keep in the freezer db(the unconfirmaed blocks are not included), 0 means no limit
}

func toTraceTable(name string) string {
	return name + "_traces"
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
		kvpath = path.Join(config.Path, "kvdb")
		frpath = path.Join(config.Path, "frdb")
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
		return nil, fmt.Errorf("failed to read the frozen blocks from the freezer db: %v", err)
	}

	l := &live{
		backend:  backend,
		kvdb:     kvdb,
		frdb:     frdb,
		freezeCh: make(chan uint64, 100),
		stopCh:   make(chan struct{}),
		tables:   tables,
		traces:   traces,
		tracer:   t,

		enableNonceTracer: config.EnableNonceTracer,
	}

	latest := l.getFreezerTail()
	var offset uint64
	if latest == 0 {
		offset = 0
	} else {
		offset = latest - tail - frozen
	}
	log.Info("Open live tracer", "path", config.Path, "offset", offset, "tail", tail, "frozen", frozen, "latest", latest, "tables", tables)

	// Initialize the latest block number as the sum of the tail, frozen, and offset
	l.latest.Store(latest)
	l.offset.Store(offset)
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

	go l.freeze(config.MaxKeepBlocks)

	return hooks, nil
}

func (l *live) OnBlockStart(ev tracing.BlockEvent) {
	// track the latest block number
	blknum := ev.Block.NumberU64()
	l.latest.Store(blknum)
	l.hash = ev.Block.Hash()
	if ev.Finalized != nil {
		l.finalized = ev.Finalized.Number.Uint64()
	}

	// reset local cache
	txs := ev.Block.Transactions().Len()
	for name := range l.traces {
		l.traces[name] = make([]*traceResult, 0, txs)
	}

	// Save the earliest arrived blknum as the offset only if offset was not set before
	if swapped := l.offset.CompareAndSwap(0, blknum); swapped {
		log.Info("Set live tracer offset to new head", "blknum", blknum)
	}
}

func (l *live) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	if l.enableNonceTracer {
		key := append(from.Bytes(), encodeNumber(tx.Nonce())...)
		val := tx.Hash().Bytes()
		if err := l.kvdb.Put(key, val); err != nil {
			log.Warn("Failed to put nonce into kvdb", "err", err)
		}
	}
	l.tracer.OnTxStart(env, tx, from)
}

func (l *live) OnTxEnd(receipt *types.Receipt, err error) {
	l.tracer.OnTxEnd(receipt, err)

	for name, tt := range l.tracer.Tracers() {
		trace := &traceResult{}
		result, err := tt.GetResult()
		if err != nil {
			log.Error("Failed to get tracer results", "number", l.latest.Load(), "error", err)
			trace.Error = err.Error()
		} else {
			trace.Result = result
		}
		l.traces[name] = append(l.traces[name], trace)
	}
}

func (l *live) OnBlockEnd(err error) {
	if err != nil {
		log.Warn("OnBlockEnd", "latest", l.latest.Load(), "error", err)
	}
	batch := l.kvdb.NewBatch()

	number := l.latest.Load()
	hash := l.hash
	for name, traces := range l.traces {
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
	case l.freezeCh <- l.finalized:
	default:
		// Channel is full, log a warning
		log.Warn("Block channel is full, skipping finalized block notification")
	}
}

func (l *live) readBlockTraces(ctx context.Context, name string, blknum uint64) ([]*traceResult, error) {
	if blknum > l.latest.Load() {
		return nil, errors.New("notfound")
	}
	if blknum < l.offset.Load() {
		return nil, errors.New("historical data not available")
	}

	tail := l.getFreezerTail()

	// Determine whether to read from kvdb or frdb
	var (
		data []byte
		err  error
	)
	if blknum >= tail {
		// Data is in kvdb
		data, err = l.readFromKVDB(ctx, name, blknum)
	} else {
		// Data is in frdb
		data, err = l.readFromFRDB(name, blknum)
	}
	if err != nil {
		return nil, err
	}

	var traces []*traceResult
	err = rlp.DecodeBytes(data, &traces)
	return traces, err
}

func (l *live) readFromKVDB(ctx context.Context, name string, blknum uint64) ([]byte, error) {
	header, err := l.backend.HeaderByNumber(ctx, rpc.BlockNumber(blknum))
	if err != nil {
		return nil, err
	}

	kvKey := toKVKey(name, blknum, header.Hash())
	data, err := l.kvdb.Get(kvKey)
	if err != nil {
		return nil, fmt.Errorf("traces not found in kvdb for block %d: %w", blknum, err)
	}
	return data, err
}

func (l *live) readFromFRDB(name string, blknum uint64) ([]byte, error) {
	table := toTraceTable(name)
	var data []byte
	err := l.frdb.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		var err error
		data, err = reader.Ancient(table, blknum-l.offset.Load())
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("traces not found in frdb for block %d: %w", blknum, err)
	}
	return data, nil
}

// Close the frdb and kvdb
// TODO: when to close it?
func (l *live) Close() {
	close(l.stopCh)

	if err := l.kvdb.Close(); err != nil {
		log.Error("Close kvdb failed", "err", err)
	}

	if err := l.frdb.Close(); err != nil {
		log.Error("Close freeze db failed", "err", err)
	}
}
