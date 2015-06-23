package api

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/docserver"
	"github.com/ethereum/go-ethereum/common/natspec"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
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
		"admin_addPeer":            (*adminApi).AddPeer,
		"admin_peers":              (*adminApi).Peers,
		"admin_nodeInfo":           (*adminApi).NodeInfo,
		"admin_exportChain":        (*adminApi).ExportChain,
		"admin_importChain":        (*adminApi).ImportChain,
		"admin_verbosity":          (*adminApi).Verbosity,
		"admin_chainSyncStatus":    (*adminApi).ChainSyncStatus,
		"admin_setSolc":            (*adminApi).SetSolc,
		"admin_datadir":            (*adminApi).DataDir,
		"admin_startRPC":           (*adminApi).StartRPC,
		"admin_stopRPC":            (*adminApi).StopRPC,
		"admin_setGlobalRegistrar": (*adminApi).SetGlobalRegistrar,
		"admin_setHashReg":         (*adminApi).SetHashReg,
		"admin_setUrlHint":         (*adminApi).SetUrlHint,
		"admin_saveInfo":           (*adminApi).SaveInfo,
		"admin_register":           (*adminApi).Register,
		"admin_registerUrl":        (*adminApi).RegisterUrl,
		"admin_startNatSpec":       (*adminApi).StartNatSpec,
		"admin_stopNatSpec":        (*adminApi).StopNatSpec,
		"admin_getContractInfo":    (*adminApi).GetContractInfo,
		"admin_httpGet":            (*adminApi).HttpGet,
	}
)

// admin callback handler
type adminhandler func(*adminApi, *shared.Request) (interface{}, error)

// admin api provider
type adminApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	codec    codec.Codec
	coder    codec.ApiCoder
	ds       *docserver.DocServer
}

// create a new admin api instance
func NewAdminApi(xeth *xeth.XEth, ethereum *eth.Ethereum, codec codec.Codec) *adminApi {
	return &adminApi{
		xeth:     xeth,
		ethereum: ethereum,
		codec:    codec,
		coder:    codec.New(nil),
		ds:       docserver.New("/"),
	}
}

// collection with supported methods
func (self *adminApi) Methods() []string {
	methods := make([]string, len(AdminMapping))
	i := 0
	for k := range AdminMapping {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *adminApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := AdminMapping[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *adminApi) Name() string {
	return shared.AdminApiName
}

func (self *adminApi) ApiVersion() string {
	return AdminApiversion
}

func (self *adminApi) AddPeer(req *shared.Request) (interface{}, error) {
	args := new(AddPeerArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
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
	if err := self.coder.Decode(req.Params, &args); err != nil {
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
	if err := self.coder.Decode(req.Params, &args); err != nil {
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
	if err := self.coder.Decode(req.Params, &args); err != nil {
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
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	solc, err := self.xeth.SetSolc(args.Path)
	if err != nil {
		return nil, err
	}
	return solc.Info(), nil
}

func (self *adminApi) StartRPC(req *shared.Request) (interface{}, error) {
	args := new(StartRPCArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	cfg := comms.HttpConfig{
		ListenAddress: args.ListenAddress,
		ListenPort:    args.ListenPort,
		CorsDomain:    args.CorsDomain,
	}

	apis, err := ParseApiString(args.Apis, self.codec, self.xeth, self.ethereum)
	if err != nil {
		return false, err
	}

	err = comms.StartHttp(cfg, self.codec, Merge(apis...))
	if err == nil {
		return true, nil
	}
	return false, err
}

func (self *adminApi) StopRPC(req *shared.Request) (interface{}, error) {
	comms.StopHttp()
	return true, nil
}

func (self *adminApi) SleepBlocks(req *shared.Request) (interface{}, error) {
	args := new(SleepBlocksArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	var timer <-chan time.Time
	var height *big.Int
	var err error
	if args.Timeout > 0 {
		timer = time.NewTimer(time.Duration(args.Timeout) * time.Second).C
	}

	height = new(big.Int).Add(self.xeth.CurrentBlock().Number(), big.NewInt(args.N))
	height, err = sleepBlocks(self.xeth.UpdateState(), height, timer)
	if err != nil {
		return nil, err
	}
	return height.Uint64(), nil
}

func sleepBlocks(wait chan *big.Int, height *big.Int, timer <-chan time.Time) (newHeight *big.Int, err error) {
	wait <- height
	select {
	case <-timer:
		// if times out make sure the xeth loop does not block
		go func() {
			select {
			case wait <- nil:
			case <-wait:
			}
		}()
		return nil, fmt.Errorf("timeout")
	case newHeight = <-wait:
	}
	return
}

// 	sec, err := call.Argument(0).ToInteger()
// 	if err != nil {
// 		fmt.Println(err)
// 		return otto.FalseValue()
// 	}
// 	time.Sleep(time.Duration(sec) * time.Second)
// 	return otto.UndefinedValue()
// }
func (self *adminApi) SetGlobalRegistrar(req *shared.Request) (interface{}, error) {
	args := new(SetGlobalRegistrarArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	sender := common.HexToAddress(args.ContractAddress)

	reg := registrar.New(self.xeth)
	err := reg.SetGlobalRegistrar(args.NameReg, sender)
	if err != nil {
		return false, err
	}

	return registrar.GlobalRegistrarAddr, nil
}

func (self *adminApi) SetHashReg(req *shared.Request) (interface{}, error) {
	args := new(SetHashRegArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	reg := registrar.New(self.xeth)
	sender := common.HexToAddress(args.Sender)
	err := reg.SetHashReg(args.HashReg, sender)
	if err != nil {
		return false, err
	}

	return registrar.HashRegAddr, nil
}

func (self *adminApi) SetUrlHint(req *shared.Request) (interface{}, error) {
	args := new(SetUrlHintArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	urlHint := args.UrlHint
	sender := common.HexToAddress(args.Sender)

	reg := registrar.New(self.xeth)
	err := reg.SetUrlHint(urlHint, sender)
	if err != nil {
		return nil, err
	}

	return registrar.UrlHintAddr, nil
}

func (self *adminApi) SaveInfo(req *shared.Request) (interface{}, error) {
	args := new(SaveInfoArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	contenthash, err := compiler.SaveInfo(&args.ContractInfo, args.Filename)
	if err != nil {
		return nil, err
	}

	return contenthash.Hex(), nil
}

func (self *adminApi) Register(req *shared.Request) (interface{}, error) {
	args := new(RegisterArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	sender := common.HexToAddress(args.Sender)
	// sender and contract address are passed as hex strings
	codeb := self.xeth.CodeAtBytes(args.Address)
	codeHash := common.BytesToHash(crypto.Sha3(codeb))
	contentHash := common.HexToHash(args.ContentHashHex)
	registry := registrar.New(self.xeth)

	_, err := registry.SetHashToHash(sender, codeHash, contentHash)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (self *adminApi) RegisterUrl(req *shared.Request) (interface{}, error) {
	args := new(RegisterUrlArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	sender := common.HexToAddress(args.Sender)
	registry := registrar.New(self.xeth)
	_, err := registry.SetUrlToHash(sender, common.HexToHash(args.ContentHash), args.Url)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (self *adminApi) StartNatSpec(req *shared.Request) (interface{}, error) {
	self.ethereum.NatSpec = true
	return true, nil
}

func (self *adminApi) StopNatSpec(req *shared.Request) (interface{}, error) {
	self.ethereum.NatSpec = false
	return true, nil
}

func (self *adminApi) GetContractInfo(req *shared.Request) (interface{}, error) {
	args := new(GetContractInfoArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	infoDoc, err := natspec.FetchDocsForContract(args.Contract, self.xeth, self.ds)
	if err != nil {
		return nil, err
	}

	var info interface{}
	err = self.coder.Decode(infoDoc, &info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (self *adminApi) HttpGet(req *shared.Request) (interface{}, error) {
	args := new(HttpGetArgs)
	if err := self.coder.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	resp, err := self.ds.Get(args.Uri, args.Path)
	if err != nil {
		return nil, err
	}

	return string(resp), nil
}
