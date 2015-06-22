package api

import (
	"fmt"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	DebugApiVersion = "1.0"
)

var (
	// mapping between methods and handlers
	DebugMapping = map[string]debughandler{
		"debug_dumpBlock":    (*debugApi).DumpBlock,
		"debug_getBlockRlp":  (*debugApi).GetBlockRlp,
		"debug_printBlock":   (*debugApi).PrintBlock,
		"debug_processBlock": (*debugApi).ProcessBlock,
		"debug_seedHash":     (*debugApi).SeedHash,
		"debug_setHead":      (*debugApi).SetHead,
	}
)

// debug callback handler
type debughandler func(*debugApi, *shared.Request) (interface{}, error)

// admin api provider
type debugApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]debughandler
	codec    codec.ApiCoder
}

// create a new debug api instance
func NewDebugApi(xeth *xeth.XEth, ethereum *eth.Ethereum, coder codec.Codec) *debugApi {
	return &debugApi{
		xeth:     xeth,
		ethereum: ethereum,
		methods:  DebugMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *debugApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *debugApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *debugApi) Name() string {
	return shared.DebugApiName
}

func (self *debugApi) ApiVersion() string {
	return DebugApiVersion
}

func (self *debugApi) PrintBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	return fmt.Sprintf("%s", block), nil
}

func (self *debugApi) DumpBlock(req *shared.Request) (interface{}, error) {
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

	return stateDb.RawDump(), nil
}

func (self *debugApi) GetBlockRlp(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}
	encoded, err := rlp.EncodeToBytes(block)
	return fmt.Sprintf("%x", encoded), err
}

func (self *debugApi) SetHead(req *shared.Request) (interface{}, error) {
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

func (self *debugApi) ProcessBlock(req *shared.Request) (interface{}, error) {
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

func (self *debugApi) SeedHash(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if hash, err := ethash.GetSeedHash(uint64(args.BlockNumber)); err == nil {
		return fmt.Sprintf("0x%x", hash), nil
	} else {
		return nil, err
	}
}
