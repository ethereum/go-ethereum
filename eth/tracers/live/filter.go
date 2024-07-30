package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"

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
	tracersTable = "tracers"
	tableSize    = 2 * 1024 * 1024 * 1024
)

type filter struct {
	db         *rawdb.Freezer
	traces     []json.RawMessage
	tracer     *tracers.Tracer
	latest     uint64
	offset     uint64
	once       sync.Once
	offsetFile string
}

type filterTracerConfig struct {
	Path   string          `json:"path"` // Path to the directory where the tracer logs will be stored
	Config json.RawMessage `json:"config"`
}

func newFilter(cfg json.RawMessage) (*tracing.Hooks, []rpc.API, error) {
	var config filterTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, nil, fmt.Errorf("failed to parse config: %v", err)
		}
	}
	if config.Path == "" {
		return nil, nil, errors.New("filter tracer output path is required")
	}

	db, err := rawdb.NewFreezer(config.Path, "trace", false, tableSize, map[string]bool{tracersTable: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace freezer db: %v", err)
	}

	t, err := native.NewMuxTracer(nil, config.Config)
	if err != nil {
		return nil, nil, err
	}

	tail, err := db.Tail()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the tail block number from the freezer db: %v", err)
	}
	frozen, err := db.Ancients()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the frozen block numbers from the freezer db: %v", err)
	}

	f := &filter{db: db, tracer: t, latest: tail + frozen, offsetFile: path.Join(config.Path, "OFFSET")}
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

	f.offset = uint64(offset)
	return &tracing.Hooks{
		OnBlockStart:   f.OnBlockStart,
		OnBlockEnd:     f.OnBlockEnd,
		OnGenesisBlock: f.OnGenesisBlock,
		OnTxStart:      f.OnTxStart,
		OnTxEnd:        f.OnTxEnd,

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
	}, nil, nil
}

func (f *filter) OnBlockStart(ev tracing.BlockEvent) {
	blknum := ev.Block.NumberU64()

	// save the earliest arrived blknum as the offset
	f.once.Do(func() {
		if _, err := os.Stat(f.offsetFile); err != nil && os.IsNotExist(err) {
			f.offset = blknum
			os.WriteFile(f.offsetFile, []byte(fmt.Sprintf("%d", blknum)), 0666)
		}
	})

	// truncate the freezer db if the block number is less than the latest
	if blknum >= f.latest {
		if _, err := f.db.TruncateHead(blknum - f.offset); err != nil {
			log.Error("failed to truncate filter tracer db", "error", err)
			// TODO: how to handle this error?
			return
		}
	}

	f.latest = blknum
	f.traces = make([]json.RawMessage, ev.Block.Transactions().Len())
}

func (f *filter) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	f.tracer.OnTxStart(env, tx, from)
}

func (f *filter) OnTxEnd(receipt *types.Receipt, err error) {
	f.tracer.OnTxEnd(receipt, err)

	result, err := f.tracer.GetResult()
	if err != nil {
		log.Error("failed to get tracer results", "number", f.latest, "error", err)
		return
	}
	f.traces = append(f.traces, result)
}

func (f *filter) OnBlockEnd(err error) {
	data, _ := json.Marshal(f.traces)
	f.appendData(data)
}

func (f *filter) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	f.appendData([]byte{})
}

func (f *filter) appendData(data []byte) {
	_, err := f.db.ModifyAncients(func(w ethdb.AncientWriteOp) error {
		return w.AppendRaw(tracersTable, f.latest-f.offset, data)
	})
	if err != nil {
		log.Error("write to freezer db failed", "error", err)
	}
}
