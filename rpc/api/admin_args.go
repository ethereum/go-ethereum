package api

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type AddPeerArgs struct {
	Url string
}

func (args *AddPeerArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected enode as argument")
	}

	urlstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("url", "not a string")
	}
	args.Url = urlstr

	return nil
}

type ImportExportChainArgs struct {
	Filename string
}

func (args *ImportExportChainArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected filename as argument")
	}

	filename, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("filename", "not a string")
	}
	args.Filename = filename

	return nil
}

type VerbosityArgs struct {
	Level int
}

func (args *VerbosityArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected enode as argument")
	}

	level, err := numString(obj[0])
	if err == nil {
		args.Level = int(level.Int64())
	}

	return nil
}

type SetSolcArgs struct {
	Path string
}

func (args *SetSolcArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected path as argument")
	}

	if pathstr, ok := obj[0].(string); ok {
		args.Path = pathstr
		return nil
	}

	return shared.NewInvalidTypeError("path", "not a string")
}
