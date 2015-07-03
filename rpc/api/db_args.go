package api

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

type DbArgs struct {
	Database string
	Key      string
	Value    []byte
}

func (args *DbArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	var objstr string
	var ok bool

	if objstr, ok = obj[0].(string); !ok {
		return shared.NewInvalidTypeError("database", "not a string")
	}
	args.Database = objstr

	if objstr, ok = obj[1].(string); !ok {
		return shared.NewInvalidTypeError("key", "not a string")
	}
	args.Key = objstr

	if len(obj) > 2 {
		objstr, ok = obj[2].(string)
		if !ok {
			return shared.NewInvalidTypeError("value", "not a string")
		}

		args.Value = []byte(objstr)
	}

	return nil
}

func (a *DbArgs) requirements() error {
	if len(a.Database) == 0 {
		return shared.NewValidationError("Database", "cannot be blank")
	}
	if len(a.Key) == 0 {
		return shared.NewValidationError("Key", "cannot be blank")
	}
	return nil
}

type DbHexArgs struct {
	Database string
	Key      string
	Value    []byte
}

func (args *DbHexArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	var objstr string
	var ok bool

	if objstr, ok = obj[0].(string); !ok {
		return shared.NewInvalidTypeError("database", "not a string")
	}
	args.Database = objstr

	if objstr, ok = obj[1].(string); !ok {
		return shared.NewInvalidTypeError("key", "not a string")
	}
	args.Key = objstr

	if len(obj) > 2 {
		objstr, ok = obj[2].(string)
		if !ok {
			return shared.NewInvalidTypeError("value", "not a string")
		}

		args.Value = common.FromHex(objstr)
	}

	return nil
}

func (a *DbHexArgs) requirements() error {
	if len(a.Database) == 0 {
		return shared.NewValidationError("Database", "cannot be blank")
	}
	if len(a.Key) == 0 {
		return shared.NewValidationError("Key", "cannot be blank")
	}
	return nil
}
