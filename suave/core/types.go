package suave

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	builder "github.com/ethereum/go-ethereum/suave/builder/api"
)

var AllowedPeekerAny = common.HexToAddress("0xC8df3686b4Afb2BB53e60EAe97EF043FE03Fb829") // "*"

type Bytes = hexutil.Bytes

type BuildBlockArgs = types.BuildBlockArgs

type ConfidentialEthBackend interface {
	BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
	BuildEthBlockFromBundles(ctx context.Context, args *BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)
	Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error)

	builder.API
}
