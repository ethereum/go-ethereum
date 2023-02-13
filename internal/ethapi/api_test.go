package ethapi

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestTransaction_RoundTripRpcJSON(t *testing.T) {
	addr := common.HexToAddress("0x1234")
	config := params.AllEthashProtocolChanges
	signer := types.LatestSigner(config)
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	tests := allTransactionTypes(addr, config)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := types.SignNewTx(key, signer, tt.inner)
			require.NoError(t, err, "signing failed: %v", err)
			rpcTx := newRPCTransaction(tx, common.Hash{}, 1234, 9, big.NewInt(10), &params.ChainConfig{})
			require.NoError(t, err, "newRPCTransaction failed: %v", err)
			data, err := json.Marshal(rpcTx)
			require.NoError(t, err, "marshal failed: %v", err)

			got := &types.Transaction{}
			err = got.UnmarshalJSON(data)
			require.NoError(t, err, "unmarshal failed: %v", err)

			require.Equal(t, rpcTx.Hash, got.Hash(), "transaction changed after round trip")
		})
	}
}

func allTransactionTypes(addr common.Address, config *params.ChainConfig) []struct {
	name  string
	inner types.TxData
} {
	return []struct {
		name  string
		inner types.TxData
	}{
		{
			name: "LegacyTx",
			inner: &types.LegacyTx{
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       &addr,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				V:        big.NewInt(9),
				R:        big.NewInt(10),
				S:        big.NewInt(11),
			},
		},
		{
			name: "LegacyTxContractCreation",
			inner: &types.LegacyTx{
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       nil,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				V:        big.NewInt(32),
				R:        big.NewInt(10),
				S:        big.NewInt(11),
			},
		},
		{
			name: "AccessListTx",
			inner: &types.AccessListTx{
				ChainID:    config.ChainID,
				Nonce:      5,
				GasPrice:   big.NewInt(6),
				Gas:        7,
				To:         &addr,
				Value:      big.NewInt(8),
				Data:       []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{},
			},
		},
		{
			name: "AccessListTxContractCreation",
			inner: &types.AccessListTx{
				ChainID:    config.ChainID,
				Nonce:      5,
				GasPrice:   big.NewInt(6),
				Gas:        7,
				To:         nil,
				Value:      big.NewInt(8),
				Data:       []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{},
			},
		},
		{
			name: "DynamicFeeTx",
			inner: &types.DynamicFeeTx{
				ChainID:    config.ChainID,
				Nonce:      5,
				GasTipCap:  big.NewInt(6),
				GasFeeCap:  big.NewInt(9),
				Gas:        7,
				To:         &addr,
				Value:      big.NewInt(8),
				Data:       []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{},
			},
		},
		{
			name: "DynamicFeeTxContractCreation",
			inner: &types.DynamicFeeTx{
				ChainID:    config.ChainID,
				Nonce:      5,
				GasTipCap:  big.NewInt(6),
				GasFeeCap:  big.NewInt(9),
				Gas:        7,
				To:         nil,
				Value:      big.NewInt(8),
				Data:       []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{},
			},
		},
	}
}
