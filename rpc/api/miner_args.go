package api
import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"math/big"
	"fmt"
)

type StartMinerArgs struct {
	Threads int `json:"threads"`
}

func (args *StartMinerArgs) UnmarshalJSON(b []byte) (err error) {
	fmt.Printf("b=%s\n", string(b))
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) == 0 {
		args.Threads = -1
		return nil
	}

	var arg0 *big.Int
	if arg0, err = numString(obj[0]); err != nil {
		return err
	}

	if arg0.Int64() >= 0 && arg0.Int64() <= 256 {
		args.Threads = int(arg0.Int64())
	}

	return shared.NewValidationError("threads", "Must be in range [0...256]")
}

type SetExtraArgs struct {
	Data string `json:"data"`
}

type GasPriceArgs struct {
	Price string `json:"price"`
}

type MakeDAGArgs struct {
	BlockNumber uint64 `json:"blockNumber"`
}
