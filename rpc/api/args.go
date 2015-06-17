package api

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

type CompileArgs struct {
	Source string
}

func (args *CompileArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}
	argstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("arg0", "is not a string")
	}
	args.Source = argstr

	return nil
}

type FilterStringArgs struct {
	Word string
}

func (args *FilterStringArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	var argstr string
	argstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("filter", "not a string")
	}
	switch argstr {
	case "latest", "pending":
		break
	default:
		return shared.NewValidationError("Word", "Must be `latest` or `pending`")
	}
	args.Word = argstr
	return nil
}