package api

import (
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	AdminApiversion = "1.0"
	importBatchSize = 2500
)

var (
	// mapping between methods and handlers
	AdminMapping = map[string]adminhandler{
		//		"admin_startRPC": (*adminApi).StartRPC,
		//		"admin_stopRPC":  (*adminApi).StopRPC,
		"admin_addPeer":         (*adminApi).AddPeer,
		"admin_peers":           (*adminApi).Peers,
		"admin_nodeInfo":        (*adminApi).NodeInfo,
		"admin_exportChain":     (*adminApi).ExportChain,
		"admin_importChain":     (*adminApi).ImportChain,
		"admin_verbosity":       (*adminApi).Verbosity,
		"admin_chainSyncStatus": (*adminApi).ChainSyncStatus,
		"admin_setSolc":         (*adminApi).SetSolc,
		"admin_datadir":         (*adminApi).DataDir,
	}
)

// admin callback handler
type adminhandler func(*adminApi, *shared.Request) (interface{}, error)

// admin api provider
type adminApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]adminhandler
	codec    codec.ApiCoder
}

// create a new admin api instance
func NewAdminApi(xeth *xeth.XEth, ethereum *eth.Ethereum, coder codec.Codec) *adminApi {
	return &adminApi{
		xeth:     xeth,
		ethereum: ethereum,
		methods:  AdminMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *adminApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *adminApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *adminApi) Name() string {
	return AdminApiName
}

func (self *adminApi) ApiVersion() string {
	return AdminApiversion
}

func (self *adminApi) AddPeer(req *shared.Request) (interface{}, error) {
	args := new(AddPeerArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	err := self.ethereum.AddPeer(args.Url)
	if err == nil {
		return true, nil
	}
	return false, err
}

func (self *adminApi) Peers(req *shared.Request) (interface{}, error) {
	return self.ethereum.PeersInfo(), nil
}

func (self *adminApi) StartRPC(req *shared.Request) (interface{}, error) {
	return false, nil
	//	Enable when http rpc interface is refactored to prevent import cycles
	//	args := new(StartRpcArgs)
	//	if err := self.codec.Decode(req.Params, &args); err != nil {
	//		return nil, shared.NewDecodeParamError(err.Error())
	//	}
	//
	//	cfg := rpc.RpcConfig{
	//		ListenAddress: args.Address,
	//		ListenPort:    args.Port,
	//	}
	//
	//	err := rpc.Start(self.xeth, cfg)
	//	if err == nil {
	//		return true, nil
	//	}
	//	return false, err
}

func (self *adminApi) StopRPC(req *shared.Request) (interface{}, error) {
	return false, nil
	//	Enable when http rpc interface is refactored to prevent import cycles
	//	rpc.Stop()
	//	return true, nil
}

func (self *adminApi) NodeInfo(req *shared.Request) (interface{}, error) {
	return self.ethereum.NodeInfo(), nil
}

func (self *adminApi) DataDir(req *shared.Request) (interface{}, error) {
	return self.ethereum.DataDir, nil
}

func hasAllBlocks(chain *core.ChainManager, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}
	return true
}

func (self *adminApi) ImportChain(req *shared.Request) (interface{}, error) {
	args := new(ImportExportChainArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	fh, err := os.Open(args.Filename)
	if err != nil {
		return false, err
	}
	defer fh.Close()
	stream := rlp.NewStream(fh, 0)

	// Run actual the import.
	blocks := make(types.Blocks, importBatchSize)
	n := 0
	for batch := 0; ; batch++ {

		i := 0
		for ; i < importBatchSize; i++ {
			var b types.Block
			if err := stream.Decode(&b); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("at block %d: %v", n, err)
			}
			blocks[i] = &b
			n++
		}
		if i == 0 {
			break
		}
		// Import the batch.
		if hasAllBlocks(self.ethereum.ChainManager(), blocks[:i]) {
			continue
		}
		if _, err := self.ethereum.ChainManager().InsertChain(blocks[:i]); err != nil {
			return false, fmt.Errorf("invalid block %d: %v", n, err)
		}
	}
	return true, nil
}

func (self *adminApi) ExportChain(req *shared.Request) (interface{}, error) {
	args := new(ImportExportChainArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	fh, err := os.OpenFile(args.Filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer fh.Close()
	if err := self.ethereum.ChainManager().Export(fh); err != nil {
		return false, err
	}

	return true, nil
}

func (self *adminApi) Verbosity(req *shared.Request) (interface{}, error) {
	args := new(VerbosityArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	glog.SetV(args.Level)
	return true, nil
}

func (self *adminApi) ChainSyncStatus(req *shared.Request) (interface{}, error) {
	pending, cached, importing, estimate := self.ethereum.Downloader().Stats()

	return map[string]interface{}{
		"blocksAvailable":        pending,
		"blocksWaitingForImport": cached,
		"importing":              importing,
		"estimate":               estimate.String(),
	}, nil
}

func (self *adminApi) SetSolc(req *shared.Request) (interface{}, error) {
	args := new(SetSolcArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	solc, err := self.xeth.SetSolc(args.Path)
	if err != nil {
		return nil, err
	}
	return solc.Info(), nil
}
