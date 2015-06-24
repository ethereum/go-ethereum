package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type WaitForBlockArgs struct {
	MinHeight int
	Timeout   int // in seconds
}

func (args *WaitForBlockArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) > 2 {
		return fmt.Errorf("waitForArgs needs 0, 1, 2 arguments")
	}

	// default values when not provided
	args.MinHeight = -1
	args.Timeout = -1

	if len(obj) >= 1 {
		var minHeight *big.Int
		if minHeight, err = numString(obj[0]); err != nil {
			return err
		}
		args.MinHeight = int(minHeight.Int64())
	}

	if len(obj) >= 2 {
		timeout, err := numString(obj[1])
		if err != nil {
			return err
		}
		args.Timeout = int(timeout.Int64())
	}

	return nil
}

type MetricsArgs struct {
	Raw bool
}

func (args *MetricsArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}
	if len(obj) > 1 {
		return fmt.Errorf("metricsArgs needs 0, 1 arguments")
	}
	// default values when not provided
	if len(obj) >= 1 && obj[0] != nil {
		if value, ok := obj[0].(bool); !ok {
			return fmt.Errorf("invalid argument %v", reflect.TypeOf(obj[0]))
		} else {
			args.Raw = value
		}
	}
	return nil
}
