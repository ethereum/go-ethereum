package api

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	DbApiversion = "1.0"
)

var (
	// mapping between methods and handlers
	DbMapping = map[string]dbhandler{
		"db_getString": (*dbApi).GetString,
		"db_putString": (*dbApi).PutString,
		"db_getHex":    (*dbApi).GetHex,
		"db_putHex":    (*dbApi).PutHex,
	}
)

// db callback handler
type dbhandler func(*dbApi, *shared.Request) (interface{}, error)

// db api provider
type dbApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]dbhandler
	codec    codec.ApiCoder
}

// create a new db api instance
func NewDbApi(xeth *xeth.XEth, ethereum *eth.Ethereum, coder codec.Codec) *dbApi {
	return &dbApi{
		xeth:     xeth,
		ethereum: ethereum,
		methods:  DbMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *dbApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *dbApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *dbApi) Name() string {
	return shared.DbApiName
}

func (self *dbApi) ApiVersion() string {
	return DbApiversion
}

func (self *dbApi) GetString(req *shared.Request) (interface{}, error) {
	args := new(DbArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if err := args.requirements(); err != nil {
		return nil, err
	}

	ret, err := self.xeth.DbGet([]byte(args.Database + args.Key))
	return string(ret), err
}

func (self *dbApi) PutString(req *shared.Request) (interface{}, error) {
	args := new(DbArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if err := args.requirements(); err != nil {
		return nil, err
	}

	return self.xeth.DbPut([]byte(args.Database+args.Key), args.Value), nil
}

func (self *dbApi) GetHex(req *shared.Request) (interface{}, error) {
	args := new(DbHexArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if err := args.requirements(); err != nil {
		return nil, err
	}

	if res, err := self.xeth.DbGet([]byte(args.Database + args.Key)); err == nil {
		return newHexData(res), nil
	} else {
		return nil, err
	}
}

func (self *dbApi) PutHex(req *shared.Request) (interface{}, error) {
	args := new(DbHexArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if err := args.requirements(); err != nil {
		return nil, err
	}

	return self.xeth.DbPut([]byte(args.Database+args.Key), args.Value), nil
}
