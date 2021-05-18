package eth

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/sha3"
	"hash"
	"math/big"
)

type helpHash struct {
	hashed hash.Hash
}

func newHash() *helpHash {

	return &helpHash{hashed: sha3.NewLegacyKeccak256()}
}

func (h *helpHash) Reset() {
	h.hashed.Reset()
}

func (h *helpHash) Update(key, val []byte) {
	h.hashed.Write(key)
	h.hashed.Write(val)
}

func (h *helpHash) Hash() common.Hash {
	return common.BytesToHash(h.hashed.Sum(nil))
}

type PreExecTx struct {
	ChainId  *big.Int
	From     string
	To       string
	Data     string
	Value    string
	Gas      string
	GasPrice string
	Nonce    string
}

// PreExecAPI provides pre exec info for rpc
type PreExecAPI struct {
	e *Ethereum
}

func NewPreExecAPI(e *Ethereum) *PreExecAPI {
	return &PreExecAPI{e: e}
}

func (api *PreExecAPI) GetBlockAndMsg(origin *PreExecTx, number *big.Int) (*types.Block, *types.Message) {
	fromAddr := common.HexToAddress(origin.From)
	toAddr := common.HexToAddress(origin.To)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    hexutil.MustDecodeUint64(origin.Nonce),
		To:       &toAddr,
		Value:    hexutil.MustDecodeBig(origin.Value),
		Gas:      hexutil.MustDecodeUint64(origin.Gas),
		GasPrice: hexutil.MustDecodeBig(origin.GasPrice),
		Data:     hexutil.MustDecode(origin.Data),
	})

	number.Add(number, big.NewInt(1))
	block := types.NewBlock(
		&types.Header{Number: number},
		[]*types.Transaction{tx}, nil, nil, newHash())

	msg := types.NewMessage(
		fromAddr,
		&toAddr,
		hexutil.MustDecodeUint64(origin.Nonce),
		hexutil.MustDecodeBig(origin.Value),
		hexutil.MustDecodeUint64(origin.Gas),
		hexutil.MustDecodeBig(origin.GasPrice),
		hexutil.MustDecode(origin.Data),
		nil, false,
	)

	return block, &msg
}

func (api *PreExecAPI) GetLogs(ctx context.Context, origin *PreExecTx) (*types.Receipt, error) {
	var (
		bc   = api.e.blockchain
	)
	header, err := api.e.APIBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return nil, err
	}
	latestNumber := header.Number

	parent := bc.GetBlockByNumber(latestNumber.Uint64())
	stateDb, err := state.New(parent.Header().Root, bc.StateCache(), bc.Snapshots())
	if err != nil {
		return nil, err
	}

	block, msg := api.GetBlockAndMsg(origin, latestNumber)
	tx := block.Transactions()[0]
	gas := tx.Gas()
	gp := new(core.GasPool).AddGas(gas)

	stateDb.Prepare(tx.Hash(), block.Hash(), 0)
	recept, err := core.ApplyTransactionForPreExec(
		bc.Config(), bc, nil, gp, stateDb, header, tx, *msg, &gas, *bc.GetVMConfig())
	if err != nil {
		return nil, err
	}
	log.Info("process info", "logs", recept.Logs, "used gas", recept.GasUsed, "err", err)
	return recept, nil
}

func (api *PreExecAPI) TraceTx(ctx context.Context, tx *PreExecTx) (interface{}, error) {
	return tx.From, nil
}
