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

	passhrase, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("passhrase", "not a string")
	}
	args.Passphrase = passhrase

	return nil
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

	addr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addr

	passhrase, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("passhrase", "not a string")
	}
	args.Passphrase = passhrase

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

	addrstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addrstr

	passphrasestr, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("passphrase", "not a string")
	}
	args.Passphrase = passphrasestr

	return nil
}
