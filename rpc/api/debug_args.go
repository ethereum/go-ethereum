package api
import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"math/big"
	"fmt"
)

type SeedHashArgs struct {
	Number uint64 `json:"number"`
}

type WaitForBlockArgs struct {
	MinHeight int `json:"minHeight"`
	Timeout   int `json:"timeout"`	// in seconds
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
