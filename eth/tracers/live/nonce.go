package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func init() {
	tracers.LiveDirectory.Register("nonce", newNonce)
}

type nonce struct {
	backend tracing.Backend
	kvdb    ethdb.Database
	latest  atomic.Uint64
}

type nonceTracerConfig struct {
	Path string `json:"path"` // Path to the directory where the tracer logs will be stored
}

func newNonce(cfg json.RawMessage, stack tracers.LiveApiRegister, backend tracing.Backend) (*tracing.Hooks, error) {
	var config nonceTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %v", err)
		}
	}
	if config.Path == "" {
		return nil, errors.New("nonce tracer output path is required")
	}

	kvdb, err := rawdb.NewPebbleDBDatabase(config.Path, 128, 1024, "nonce", false, false)
	if err != nil {
		return nil, err
	}

	n := &nonce{
		backend: backend,
		kvdb:    kvdb,
	}
	log.Info("Open nonce tracer", "path", config.Path)

	apis := []rpc.API{{Namespace: "eth", Service: n}}
	stack.RegisterAPIs(apis)

	return &tracing.Hooks{
		OnBlockStart: n.onBlockStart,
		OnTxStart:    n.onTxStart,
	}, nil
}

func (n *nonce) onBlockStart(ev tracing.BlockEvent) {
	blknum := ev.Block.NumberU64()
	n.latest.Store(blknum)
}

func (n *nonce) onTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	key := append(from.Bytes(), encodeNumber(tx.Nonce())...)
	val := tx.Hash().Bytes()
	if err := n.kvdb.Put(key, val); err != nil {
		log.Warn("Failed to put nonce kvdb", "err", err)
	}
}

func (n *nonce) GetTransactionBySenderAndNonce(ctx context.Context, sender common.Address, nonce hexutil.Uint) (*ethapi.RPCTransaction, error) {
	// TODO:
	// 1. return nil if sender is a contract
	// 2. check with txpool first
	txHash, err := n.kvdb.Get(append(sender.Bytes(), encodeNumber(uint64(nonce))...))
	if err != nil {
		return nil, nil
	}

	found, tx, blockHash, blockNumber, index, err := n.backend.GetTransaction(ctx, common.BytesToHash(txHash))
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("transaction not found")
	}

	header, err := n.backend.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	return ethapi.NewRPCTransaction(tx, blockHash, blockNumber, header.Time, index, header.BaseFee, n.backend.ChainConfig()), nil
}

func (n *nonce) Close() {
	if err := n.kvdb.Close(); err != nil {
		log.Error("Close kvdb failed", "err", err)
	}
}
