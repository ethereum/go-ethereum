package api

import (
	"math/big"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

var (
	// mapping between methods and handlers
	shhMapping = map[string]shhhandler{
		"shh_version":          (*shhApi).Version,
		"shh_post":             (*shhApi).Post,
		"shh_hasIdentity":      (*shhApi).HasIdentity,
		"shh_newIdentity":      (*shhApi).NewIdentity,
		"shh_newFilter":        (*shhApi).NewFilter,
		"shh_uninstallFilter":  (*shhApi).UninstallFilter,
		"shh_getFilterChanges": (*shhApi).GetFilterChanges,
	}
)

func newWhisperOfflineError(method string) error {
	return shared.NewNotAvailableError(method, "whisper offline")
}

// net callback handler
type shhhandler func(*shhApi, *shared.Request) (interface{}, error)

// shh api provider
type shhApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]shhhandler
	codec    codec.ApiCoder
}

// create a new whisper api instance
func NewShhApi(xeth *xeth.XEth, eth *eth.Ethereum, coder codec.Codec) *shhApi {
	return &shhApi{
		xeth:     xeth,
		ethereum: eth,
		methods:  shhMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *shhApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *shhApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *shhApi) Name() string {
	return ShhApiName
}

func (self *shhApi) Version(req *shared.Request) (interface{}, error) {
	w := self.xeth.Whisper()
	if w == nil {
		return nil, newWhisperOfflineError(req.Method)
	}

	return w.Version(), nil
}

func (self *shhApi) Post(req *shared.Request) (interface{}, error) {
	w := self.xeth.Whisper()
	if w == nil {
		return nil, newWhisperOfflineError(req.Method)
	}

	args := new(WhisperMessageArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	err := w.Post(args.Payload, args.To, args.From, args.Topics, args.Priority, args.Ttl)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (self *shhApi) HasIdentity(req *shared.Request) (interface{}, error) {
	w := self.xeth.Whisper()
	if w == nil {
		return nil, newWhisperOfflineError(req.Method)
	}

	args := new(WhisperIdentityArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	return w.HasIdentity(args.Identity), nil
}

func (self *shhApi) NewIdentity(req *shared.Request) (interface{}, error) {
	w := self.xeth.Whisper()
	if w == nil {
		return nil, newWhisperOfflineError(req.Method)
	}

	return w.NewIdentity(), nil
}

func (self *shhApi) NewFilter(req *shared.Request) (interface{}, error) {
	args := new(WhisperFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	id := self.xeth.NewWhisperFilter(args.To, args.From, args.Topics)
	return newHexNum(big.NewInt(int64(id)).Bytes()), nil
}

func (self *shhApi) UninstallFilter(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}
	return self.xeth.UninstallWhisperFilter(args.Id), nil
}

func (self *shhApi) GetFilterChanges(req *shared.Request) (interface{}, error) {
	w := self.xeth.Whisper()
	if w == nil {
		return nil, newWhisperOfflineError(req.Method)
	}

	// Retrieve all the new messages arrived since the last request
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	return self.xeth.WhisperMessagesChanged(args.Id), nil
}

func (self *shhApi) GetMessages(req *shared.Request) (interface{}, error) {
	w := self.xeth.Whisper()
	if w == nil {
		return nil, newWhisperOfflineError(req.Method)
	}

	// Retrieve all the cached messages matching a specific, existing filter
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, err
	}

	return self.xeth.WhisperMessages(args.Id), nil
}
