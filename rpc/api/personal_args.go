package api

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type NewAccountArgs struct {
	Passphrase string
}

func (args *NewAccountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	if passhrase, ok := obj[0].(string); ok {
		args.Passphrase = passhrase
		return nil
	}

	return shared.NewInvalidTypeError("passhrase", "not a string")
}

type DeleteAccountArgs struct {
	Address    string
	Passphrase string
}

func (args *DeleteAccountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if addr, ok := obj[0].(string); ok {
		args.Address = addr
	} else {
		return shared.NewInvalidTypeError("address", "not a string")
	}

	if passhrase, ok := obj[1].(string); ok {
		args.Passphrase = passhrase
	} else {
		return shared.NewInvalidTypeError("passhrase", "not a string")
	}

	return nil
}

type UnlockAccountArgs struct {
	Address    string
	Passphrase string
	Duration   int
}

func (args *UnlockAccountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	args.Duration = -1

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if addrstr, ok := obj[0].(string); ok {
		args.Address = addrstr
	} else {
		return shared.NewInvalidTypeError("address", "not a string")
	}

	if passphrasestr, ok := obj[1].(string); ok {
		args.Passphrase = passphrasestr
	} else {
		return shared.NewInvalidTypeError("passphrase", "not a string")
	}

	return nil
}
