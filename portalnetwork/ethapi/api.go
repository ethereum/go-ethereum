package ethapi

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/portalnetwork/history"
	"github.com/ethereum/go-ethereum/rpc"
)

var errParameterNotImplemented = errors.New("parameter not implemented")

// marshalReceipt marshals a transaction receipt into a JSON object.
func marshalReceipt(receipt *types.Receipt, blockHash common.Hash, blockNumber uint64, signer types.Signer, tx *types.Transaction, txIndex int) map[string]interface{} {
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   tx.Hash(),
		"transactionIndex":  hexutil.Uint64(txIndex),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
		"effectiveGasPrice": (*hexutil.Big)(receipt.EffectiveGasPrice),
	}

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = []*types.Log{}
	}

	if tx.Type() == types.BlobTxType {
		fields["blobGasUsed"] = hexutil.Uint64(receipt.BlobGasUsed)
		fields["blobGasPrice"] = (*hexutil.Big)(receipt.BlobGasPrice)
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields
}

type API struct {
	History *history.Network
	ChainID *big.Int
}

func (p *API) ChainId() hexutil.Uint64 {
	return (hexutil.Uint64)(p.ChainID.Uint64())
}

func (p *API) GetBlockByHash(hash *common.Hash, fullTransactions bool) (map[string]interface{}, error) {
	blockHeader, err := p.History.GetBlockHeader(hash.Bytes())
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	blockBody, err := p.History.GetBlockBody(hash.Bytes())
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	block := types.NewBlockWithHeader(blockHeader).WithBody(*blockBody)
	//static configuration of Config, currently only mainnet implemented
	return ethapi.RPCMarshalBlock(block, true, fullTransactions, params.MainnetChainConfig), nil
}

func (p *API) GetBlockReceipts(blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	hash, isHhash := blockNrOrHash.Hash()
	if !isHhash {
		return nil, errParameterNotImplemented
	}

	blockReceipts, err := p.History.GetReceipts(hash.Bytes())
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	blockBody, err := p.History.GetBlockBody(hash.Bytes())
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	blockHeader, err := p.History.GetBlockHeader(hash.Bytes())
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	txs := blockBody.Transactions
	if len(txs) != len(blockReceipts) {
		return nil, fmt.Errorf("receipts length mismatch: %d vs %d", len(txs), len(blockReceipts))
	}

	// Derive the sender.
	signer := types.MakeSigner(params.MainnetChainConfig, blockHeader.Number, blockHeader.Time)

	result := make([]map[string]interface{}, len(blockReceipts))
	for i, receipt := range blockReceipts {
		result[i] = marshalReceipt(receipt, blockHeader.Hash(), blockHeader.Number.Uint64(), signer, txs[i], i)
	}

	return result, nil
}

func (p *API) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	blockBody, err := p.History.GetBlockBody(hash.Bytes())
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	n := hexutil.Uint(len(blockBody.Transactions))
	return &n
}
