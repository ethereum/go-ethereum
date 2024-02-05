package types

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto"
)

func TestNewOrderTransactionByNonce(t *testing.T) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 1)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}

	groups := map[common.Address]OrderTransactions{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for i := 0; i < 1; i++ {
			//tx, _ := SignTx(NewTransaction(uint64(start+i), common.Address{}, big.NewInt(100), 100, big.NewInt(int64(start+i)), nil), signer, key)
			orderTx := NewOrderTransaction(uint64(start+i), big.NewInt(1), big.NewInt(2), common.Address{}, common.Address{}, common.Address{}, common.Address{}, "new", "BID", "test", common.Hash{}, 1001)

			groups[addr] = append(groups[addr], orderTx)
		}
	}
	tx := NewOrderTransactionByNonce(OrderTxSigner{}, groups)
	t.Log(tx)
}
