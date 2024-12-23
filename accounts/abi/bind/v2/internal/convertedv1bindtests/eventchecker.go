// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package convertedv1bindtests

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// TODO: convert this type to value type after everything works.
// EventCheckerMetaData contains all meta data concerning the EventChecker contract.
var EventCheckerMetaData = &bind.MetaData{
	ABI:     "[{\"type\":\"event\",\"name\":\"empty\",\"inputs\":[]},{\"type\":\"event\",\"name\":\"indexed\",\"inputs\":[{\"name\":\"addr\",\"type\":\"address\",\"indexed\":true},{\"name\":\"num\",\"type\":\"int256\",\"indexed\":true}]},{\"type\":\"event\",\"name\":\"mixed\",\"inputs\":[{\"name\":\"addr\",\"type\":\"address\",\"indexed\":true},{\"name\":\"num\",\"type\":\"int256\"}]},{\"type\":\"event\",\"name\":\"anonymous\",\"anonymous\":true,\"inputs\":[]},{\"type\":\"event\",\"name\":\"dynamic\",\"inputs\":[{\"name\":\"idxStr\",\"type\":\"string\",\"indexed\":true},{\"name\":\"idxDat\",\"type\":\"bytes\",\"indexed\":true},{\"name\":\"str\",\"type\":\"string\"},{\"name\":\"dat\",\"type\":\"bytes\"}]},{\"type\":\"event\",\"name\":\"unnamed\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"indexed\":true},{\"name\":\"\",\"type\":\"uint256\",\"indexed\":true}]}]",
	Pattern: "253d421f98e29b25315bde79c1251ab27c",
}

// EventChecker is an auto generated Go binding around an Ethereum contract.
type EventChecker struct {
	abi abi.ABI
}

// NewEventChecker creates a new instance of EventChecker.
func NewEventChecker() (*EventChecker, error) {
	parsed, err := EventCheckerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &EventChecker{abi: *parsed}, nil
}

func (_EventChecker *EventChecker) PackConstructor() []byte {
	res, _ := _EventChecker.abi.Pack("")
	return res
}

// EventCheckerDynamic represents a Dynamic event raised by the EventChecker contract.
type EventCheckerDynamic struct {
	IdxStr common.Hash
	IdxDat common.Hash
	Str    string
	Dat    []byte
	Raw    *types.Log // Blockchain specific contextual infos
}

const EventCheckerDynamicEventName = "dynamic"

func (_EventChecker *EventChecker) UnpackDynamicEvent(log *types.Log) (*EventCheckerDynamic, error) {
	event := "dynamic"
	if log.Topics[0] != _EventChecker.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(EventCheckerDynamic)
	if len(log.Data) > 0 {
		if err := _EventChecker.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _EventChecker.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// EventCheckerEmpty represents a Empty event raised by the EventChecker contract.
type EventCheckerEmpty struct {
	Raw *types.Log // Blockchain specific contextual infos
}

const EventCheckerEmptyEventName = "empty"

func (_EventChecker *EventChecker) UnpackEmptyEvent(log *types.Log) (*EventCheckerEmpty, error) {
	event := "empty"
	if log.Topics[0] != _EventChecker.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(EventCheckerEmpty)
	if len(log.Data) > 0 {
		if err := _EventChecker.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _EventChecker.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// EventCheckerIndexed represents a Indexed event raised by the EventChecker contract.
type EventCheckerIndexed struct {
	Addr common.Address
	Num  *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const EventCheckerIndexedEventName = "indexed"

func (_EventChecker *EventChecker) UnpackIndexedEvent(log *types.Log) (*EventCheckerIndexed, error) {
	event := "indexed"
	if log.Topics[0] != _EventChecker.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(EventCheckerIndexed)
	if len(log.Data) > 0 {
		if err := _EventChecker.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _EventChecker.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// EventCheckerMixed represents a Mixed event raised by the EventChecker contract.
type EventCheckerMixed struct {
	Addr common.Address
	Num  *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const EventCheckerMixedEventName = "mixed"

func (_EventChecker *EventChecker) UnpackMixedEvent(log *types.Log) (*EventCheckerMixed, error) {
	event := "mixed"
	if log.Topics[0] != _EventChecker.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(EventCheckerMixed)
	if len(log.Data) > 0 {
		if err := _EventChecker.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _EventChecker.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// EventCheckerUnnamed represents a Unnamed event raised by the EventChecker contract.
type EventCheckerUnnamed struct {
	Arg0 *big.Int
	Arg1 *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const EventCheckerUnnamedEventName = "unnamed"

func (_EventChecker *EventChecker) UnpackUnnamedEvent(log *types.Log) (*EventCheckerUnnamed, error) {
	event := "unnamed"
	if log.Topics[0] != _EventChecker.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(EventCheckerUnnamed)
	if len(log.Data) > 0 {
		if err := _EventChecker.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _EventChecker.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}
