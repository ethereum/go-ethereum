package contracts

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

const (
	HexSignMethod = "2fb1b25f"
)

// Send tx sign for block number to smart contract blockSigner.
func CreateTransactionSign(chainConfig *params.ChainConfig, pool *core.TxPool, manager *accounts.Manager, block *types.Block) {
	// Find active account.
	account := accounts.Account{}
	var wallet accounts.Wallet
	if wallets := manager.Wallets(); len(wallets) > 0 {
		wallet = wallets[0]
		if accts := wallets[0].Accounts(); len(accts) > 0 {
			account = accts[0]
		}
	}

	// Create and send tx to smart contract for sign validate block.
	blockHex := common.LeftPadBytes(block.Number().Bytes(), 32)
	data := common.Hex2Bytes(HexSignMethod)
	inputData := append(data, blockHex...)
	nonce := pool.State().GetNonce(account.Address)
	tx := types.NewTransaction(nonce, common.HexToAddress(common.BlockSigners), big.NewInt(0), 100000, big.NewInt(0), inputData)
	txSigned, err := wallet.SignTx(account, tx, chainConfig.ChainId)
	if err != nil {
		log.Error("Fail to create tx sign", "error", err)
		return
	}

	// Add tx signed to local tx pool.
	pool.AddLocal(txSigned)
}
