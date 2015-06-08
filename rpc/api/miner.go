package api

import (
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	MinerVersion = "1.0.0"
)

var (
	// mapping between methods and handlers
	MinerMapping = map[string]minerhandler{
		"miner_hashrate":     (*miner).Hashrate,
		"miner_makeDAG":      (*miner).MakeDAG,
		"miner_setExtra":     (*miner).SetExtra,
		"miner_setGasPrice":  (*miner).SetGasPrice,
		"miner_startAutoDAG": (*miner).StartAutoDAG,
		"miner_start":        (*miner).StartMiner,
		"miner_stopAutoDAG":  (*miner).StopAutoDAG,
		"miner_stop":         (*miner).StopMiner,
	}
)

// miner callback handler
type minerhandler func(*miner, *shared.Request) (interface{}, error)

// miner api provider
type miner struct {
	ethereum *eth.Ethereum
	methods  map[string]minerhandler
	codec    codec.ApiCoder
}

// create a new miner api instance
func NewMinerApi(ethereum *eth.Ethereum, coder codec.Codec) *miner {
	return &miner{
		ethereum: ethereum,
		methods:  MinerMapping,
		codec:    coder.New(nil),
	}
}

// Execute given request
func (self *miner) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

// collection with supported methods
func (self *miner) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

func (self *miner) Name() string {
	return MinerApiName
}

func (self *miner) StartMiner(req *shared.Request) (interface{}, error) {
	args := new(StartMinerArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}
	if args.Threads == -1 { // (not specified by user, use default)
		args.Threads = self.ethereum.MinerThreads
	}

	self.ethereum.StartAutoDAG()
	err := self.ethereum.StartMining(args.Threads)
	if err == nil {
		return true, nil
	}

	return false, err
}

func (self *miner) StopMiner(req *shared.Request) (interface{}, error) {
	self.ethereum.StopMining()
	return true, nil
}

func (self *miner) Hashrate(req *shared.Request) (interface{}, error) {
	return self.ethereum.Miner().HashRate(), nil
}

func (self *miner) SetExtra(req *shared.Request) (interface{}, error) {
	args := new(SetExtraArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}
	self.ethereum.Miner().SetExtra([]byte(args.Data))
	return true, nil
}

func (self *miner) SetGasPrice(req *shared.Request) (interface{}, error) {
	args := new(GasPriceArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return false, err
	}

	self.ethereum.Miner().SetGasPrice(common.String2Big(args.Price))
	return true, nil
}

func (self *miner) StartAutoDAG(req *shared.Request) (interface{}, error) {
	self.ethereum.StartAutoDAG()
	return true, nil
}

func (self *miner) StopAutoDAG(req *shared.Request) (interface{}, error) {
	self.ethereum.StopAutoDAG()
	return true, nil
}

func (self *miner) MakeDAG(req *shared.Request) (interface{}, error) {
	args := new(MakeDAGArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	if args.BlockNumber < 0 {
		return false, shared.NewValidationError("BlockNumber", "BlockNumber must be positive")
	}

	err := ethash.MakeDAG(uint64(args.BlockNumber), "")
	if err == nil {
		return true, nil
	}
	return false, err
}
