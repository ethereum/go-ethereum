package ethapi

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetRootHash returns root hash for given start and end block
func (s *BlockChainAPI) GetRootHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64) (string, error) {
	root, err := s.b.GetRootHash(ctx, starBlockNr, endBlockNr)
	if err != nil {
		return "", err
	}

	return root, nil
}

func (s *BlockChainAPI) GetBorBlockReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	return s.b.GetBorBlockReceipt(ctx, hash)
}

func (s *BlockChainAPI) GetVoteOnHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64, hash string, milestoneId string) (bool, error) {
	return s.b.GetVoteOnHash(ctx, starBlockNr, endBlockNr, hash, milestoneId)
}

//
// Bor transaction utils
//

func (s *BlockChainAPI) appendRPCMarshalBorTransaction(ctx context.Context, block *types.Block, fields map[string]interface{}, fullTx bool) map[string]interface{} {
	if block != nil {
		txHash := types.GetDerivedBorTxHash(types.BorReceiptKey(block.Number().Uint64(), block.Hash()))

		borTx, blockHash, blockNumber, txIndex, _ := s.b.GetBorBlockTransactionWithBlockHash(ctx, txHash, block.Hash())
		if borTx != nil {
			formattedTxs := fields["transactions"].([]interface{})

			if fullTx {
				marshalledTx := newRPCTransaction(borTx, blockHash, blockNumber, block.Time(), txIndex, block.BaseFee(), s.b.ChainConfig())
				// newRPCTransaction calculates hash based on RLP of the transaction data.
				// In case of bor block tx, we need simple derived tx hash (same as function argument) instead of RLP hash
				marshalledTx.Hash = txHash
				marshalledTx.ChainID = nil
				fields["transactions"] = append(formattedTxs, marshalledTx)
			} else {
				fields["transactions"] = append(formattedTxs, txHash)
			}
		}
	}

	return fields
}

// BorAPI provides an API to access Bor related information.
type BorAPI struct {
	b Backend
}

// NewBorAPI creates a new Bor protocol API.
func NewBorAPI(b Backend) *BorAPI {
	return &BorAPI{b}
}

// SendRawTransactionConditional will add the signed transaction to the transaction pool.
// The sender/bundler is responsible for signing the transaction
func (api *BorAPI) SendRawTransactionConditional(ctx context.Context, input hexutil.Bytes, options types.OptionsPIP15) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return common.Hash{}, err
	}

	currentHeader := api.b.CurrentHeader()
	currentState, _, err := api.b.StateAndHeaderByNumber(ctx, rpc.BlockNumber(currentHeader.Number.Int64()))

	if currentState == nil || err != nil {
		return common.Hash{}, err
	}

	// check block number range
	if err := currentHeader.ValidateBlockNumberOptionsPIP15(options.BlockNumberMin, options.BlockNumberMax); err != nil {
		return common.Hash{}, &rpc.OptionsValidateError{Message: "out of block range. err: " + err.Error()}
	}

	// check timestamp range
	if err := currentHeader.ValidateTimestampOptionsPIP15(options.TimestampMin, options.TimestampMax); err != nil {
		return common.Hash{}, &rpc.OptionsValidateError{Message: "out of time range. err: " + err.Error()}
	}

	// check knownAccounts length (number of slots/accounts) should be less than 1000
	if err := options.KnownAccounts.ValidateLength(); err != nil {
		return common.Hash{}, &rpc.KnownAccountsLimitExceededError{Message: "limit exceeded. err: " + err.Error()}
	}

	// check knownAccounts
	if err := currentState.ValidateKnownAccounts(options.KnownAccounts); err != nil {
		return common.Hash{}, &rpc.OptionsValidateError{Message: "storage error. err: " + err.Error()}
	}

	// put options data in Tx, to use it later while block building
	tx.PutOptions(&options)

	return SubmitTransaction(ctx, api.b, tx)
}

func (api *BorAPI) GetVoteOnHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64, hash string, milestoneId string) (bool, error) {
	return api.b.GetVoteOnHash(ctx, starBlockNr, endBlockNr, hash, milestoneId)
}
