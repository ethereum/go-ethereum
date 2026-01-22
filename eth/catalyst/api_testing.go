package catalyst

import (
	"errors"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type TestingAPI struct {
	eth *eth.Ethereum
}

func NewTestingAPI(eth *eth.Ethereum) *TestingAPI {
	return &TestingAPI{
		eth: eth,
	}
}

func RegisterTestingAPI(stack *node.Node, backend *eth.Ethereum) error {
	stack.RegisterAPIs([]rpc.API{{
		Namespace:     "testing",
		Service:       NewTestingAPI(backend),
		Authenticated: false,
	},
	})
	return nil
}

func (api *TestingAPI) BuildBlockV1(parentHash common.Hash, payloadAttributes engine.PayloadAttributes, transactions []hexutil.Bytes, extraData []byte) (*engine.ExecutionPayloadEnvelope, error) {
	if api.eth.BlockChain().CurrentBlock().Hash() != parentHash {
		return nil, errors.New("parentHash is not current head")
	}
	dec := make([][]byte, 0, len(transactions))
	for _, tx := range transactions {
		dec = append(dec, tx)
	}
	txs, err := engine.DecodeTransactions(dec)
	if err != nil {
		return nil, err
	}
	args := &miner.BuildPayloadArgs{
		Parent:       parentHash,
		Timestamp:    payloadAttributes.Timestamp,
		FeeRecipient: payloadAttributes.SuggestedFeeRecipient,
		Random:       payloadAttributes.Random,
		Withdrawals:  payloadAttributes.Withdrawals,
		BeaconRoot:   payloadAttributes.BeaconRoot,
		Transactions: txs,
		ExtraData:    extraData,
	}
	payload, err := api.eth.Miner().BuildPayload(args, false)
	if err != nil {
		log.Error("Failed to build payload", "err", err)
		return nil, err
	}
	return payload.ResolveFull(), nil
}
