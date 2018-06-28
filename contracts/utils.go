package contracts

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

const (
	HexSignMethod = "2fb1b25f"
)

type rewardLog struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

// Send tx sign for block number to smart contract blockSigner.
func CreateTransactionSign(chainConfig *params.ChainConfig, pool *core.TxPool, manager *accounts.Manager, block *types.Block) error {
	if chainConfig.Clique != nil {
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
		nonce := pool.State().GetNonce(account.Address)
		tx := CreateTxSign(block.Number(), nonce, common.HexToAddress(common.BlockSigners))
		txSigned, err := wallet.SignTx(account, tx, chainConfig.ChainId)
		if err != nil {
			log.Error("Fail to create tx sign", "error", err)
			return err
		}

		// Add tx signed to local tx pool.
		pool.AddLocal(txSigned)
	}

	return nil
}

// Create tx sign.
func CreateTxSign(blockNumber *big.Int, nonce uint64, blockSigner common.Address) *types.Transaction {
	blockHex := common.LeftPadBytes(blockNumber.Bytes(), 32)
	data := common.Hex2Bytes(HexSignMethod)
	inputData := append(data, blockHex...)
	tx := types.NewTransaction(nonce, blockSigner, big.NewInt(0), 100000, big.NewInt(0), inputData)

	return tx
}

// Get signers signed for blockNumber from blockSigner contract.
func GetSignersFromContract(addrBlockSigner common.Address, client bind.ContractBackend, blockNumber uint64) ([]common.Address, error) {
	blockSigner, err := contract.NewBlockSigner(addrBlockSigner, client)
	if err != nil {
		log.Error("Fail get instance of blockSigner", "error", err)
		return nil, err
	}
	opts := new(bind.CallOpts)
	addrs, err := blockSigner.GetSigners(opts, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		log.Error("Fail get block signers", "error", err)
		return nil, err
	}

	return addrs, nil
}

// Calculate reward for reward checkpoint.
func GetRewardForCheckpoint(blockSignerAddr common.Address, number uint64, rCheckpoint uint64, client bind.ContractBackend, totalSigner *uint64) (map[common.Address]*rewardLog, error) {
	// Not reward for singer of genesis block and only calculate reward at checkpoint block.
	startBlockNumber := number - (rCheckpoint * 2) + 1
	endBlockNumber := startBlockNumber + rCheckpoint - 1
	signers := make(map[common.Address]*rewardLog)

	for i := startBlockNumber; i <= endBlockNumber; i++ {
		addrs, err := GetSignersFromContract(blockSignerAddr, client, i)
		if err != nil {
			log.Error("Fail to get signers from smartcontract.", "error", err, "blockNumber", i)
			return nil, err
		}
		// Filter duplicate address.
		if len(addrs) > 0 {
			addrSigners := make(map[common.Address]bool)
			for _, addr := range addrs {
				if _, ok := addrSigners[addr]; !ok {
					addrSigners[addr] = true
				}
			}
			for addr := range addrSigners {
				_, exist := signers[addr]
				if exist {
					signers[addr].Sign++
				} else {
					signers[addr] = &rewardLog{1, new(big.Int)}
				}
				*totalSigner++
			}
		}
	}

	log.Info("Calculate reward at checkpoint", "startBlock", startBlockNumber, "endBlock", endBlockNumber)

	return signers, nil
}

// Calculate reward for signers.
func CalculateReward(chainReward *big.Int, signers map[common.Address]*rewardLog, totalSigner uint64) (map[common.Address]*big.Int, error) {
	resultSigners := make(map[common.Address]*big.Int)
	// Add reward for signers.
	for signer, rLog := range signers {
		// Add reward for signer.
		calcReward := new(big.Int)
		calcReward.Div(chainReward, new(big.Int).SetUint64(totalSigner))
		calcReward.Mul(calcReward, new(big.Int).SetUint64(rLog.Sign))
		rLog.Reward = calcReward

		resultSigners[signer] = calcReward
	}
	jsonSigners, err := json.Marshal(signers)
	if err != nil {
		log.Error("Fail to parse json signers", "error", err)
		return nil, err
	}
	log.Info("Signers data", "signers", string(jsonSigners), "totalSigner", totalSigner, "totalReward", chainReward)

	return resultSigners, nil
}
