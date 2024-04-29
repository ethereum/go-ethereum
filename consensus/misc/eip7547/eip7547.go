package eip7547

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func VerifyInclusionList(bc *core.BlockChain, parent *types.Block, signer types.Signer, txs []*types.Transaction) error {
	statedb, err := bc.StateAt(parent.Root())
	if err != nil {
		return err
	}
	for _, tx := range txs {
		sender, _ := signer.Sender(tx)
		nonce := statedb.GetNonce(sender)
		balance := statedb.GetBalance(sender)
		opts := &txpool.ValidationOptions{
			Config: bc.Config(),
			Accept: 0 |
				1<<types.LegacyTxType |
				1<<types.AccessListTxType |
				1<<types.DynamicFeeTxType,
			MaxSize: 128 * 1024,
			MinTip:  new(big.Int),
		}
		if err := txpool.ValidateTransaction(tx, parent.Header(), signer, opts); err != nil {
			return err
		}
		stateOpts := &txpool.ValidationOptionsWithState{
			State: statedb,
			FirstNonceGap: func(addr common.Address) uint64 {
				return statedb.GetNonce(addr) // Nonce gaps are not permitted
			},
			UsedAndLeftSlots: func(addr common.Address) (int, int) {
				return 0, math.MaxInt
			},
			ExistingExpenditure: func(addr common.Address) *big.Int {
				return new(big.Int)
			},
			ExistingCost: func(addr common.Address, nonce uint64) *big.Int {
				return new(big.Int)
			},
		}
		if err := txpool.ValidateTransactionWithState(tx, signer, stateOpts); err != nil {
			return err
		}
		statedb.SetNonce(sender, nonce+1)
		statedb.SetBalance(sender, balance.Sub(balance, uint256.MustFromBig(tx.Cost())))
	}
	return nil
}
