package ethapi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/portalnetwork/history"
)

type API struct {
	History *history.HistoryNetwork
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
