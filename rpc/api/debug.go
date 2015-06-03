package api

import (
	"fmt"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	ethereum "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	DebugVersion = "1.0.0"
)

var (
	// mapping between methods and handlers
	DebugMapping = map[string]debughandler{
		"debug_dumpBlock":    (*debug).DumpBlock,
		"debug_getBlockRlp":  (*debug).GetBlockRlp,
		"debug_printBlock":   (*debug).PrintBlock,
		"debug_processBlock": (*debug).ProcessBlock,
		"debug_seedHash":     (*debug).SeedHash,
		"debug_setHead":      (*debug).SetHead,
	}
)

// debug callback handler
type debughandler func(*debug, *shared.Request) (interface{}, error)

// admin api provider
type debug struct {
	xeth     *xeth.XEth
	ethereum ethereum.Ethereum
	methods  map[string]debughandler
	codec    codec.ApiCoder
}

// create a new debug api instance
func NewDebug(xeth *xeth.XEth, ethereum ethereum.Ethereum, coder codec.Codec) *debug {
	return &debug{
		xeth:     xeth,
		ethereum: ethereum,
		methods:  DebugMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *debug) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

func (self *debug) PrintBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.EthBlockByNumber(args.BlockNumber), nil
}

func (self *debug) DumpBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}

	stateDb := state.New(block.Root(), self.ethereum.StateDb())
	if stateDb == nil {
		return nil, nil
	}

	return stateDb.Dump(), nil
}

func (self *debug) GetBlockRlp(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}
	return rlp.EncodeToBytes(block)
}

func (self *debug) SetHead(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}

	self.ethereum.ChainManager().SetHead(block)

	return nil, nil
}

func (self *debug) ProcessBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}

	old := vm.Debug
	defer func() { vm.Debug = old }()
	vm.Debug = true

	_, err := self.ethereum.BlockProcessor().RetryProcess(block)
	if err == nil {
		return true, nil
	}
	return false, err
}

func (self *debug) SeedHash(req *shared.Request) (interface{}, error) {
	args := new(SeedHashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if hash, err := ethash.GetSeedHash(args.Number); err == nil {
		return fmt.Sprintf("0x%x", hash), nil
	} else {
		return nil, err
	}
}
