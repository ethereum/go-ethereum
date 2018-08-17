// Copyright (c) 2018 Tomochain
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package contracts

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	contractValidator "github.com/ethereum/go-ethereum/contracts/validator/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

const (
	HexSignMethod           = "e341eaa4"
	RewardMasterPercent     = 30
	RewardVoterPercent      = 60
	RewardFoundationPercent = 10
	FoudationWalletAddr     = "0x0000000000000000000000000000000000000068"
)

type rewardLog struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

// Send tx sign for block number to smart contract blockSigner.
func CreateTransactionSign(chainConfig *params.ChainConfig, pool *core.TxPool, manager *accounts.Manager, block *types.Block) error {
	if chainConfig.Posv != nil {
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
		tx := CreateTxSign(block.Number(), block.Hash(), nonce, common.HexToAddress(common.BlockSigners))
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
func CreateTxSign(blockNumber *big.Int, blockHash common.Hash, nonce uint64, blockSigner common.Address) *types.Transaction {
	data := common.Hex2Bytes(HexSignMethod)
	inputData := append(data, common.LeftPadBytes(blockNumber.Bytes(), 32)...)
	inputData = append(inputData, common.LeftPadBytes(blockHash.Bytes(), 32)...)
	tx := types.NewTransaction(nonce, blockSigner, big.NewInt(0), 200000, big.NewInt(0), inputData)

	return tx
}

// Get signers signed for blockNumber from blockSigner contract.
func GetSignersFromContract(addrBlockSigner common.Address, client bind.ContractBackend, blockHash common.Hash) ([]common.Address, error) {
	blockSigner, err := contract.NewBlockSigner(addrBlockSigner, client)
	if err != nil {
		log.Error("Fail get instance of blockSigner", "error", err)
		return nil, err
	}
	opts := new(bind.CallOpts)
	addrs, err := blockSigner.GetSigners(opts, blockHash)
	if err != nil {
		log.Error("Fail get block signers", "error", err)
		return nil, err
	}

	return addrs, nil
}

// Calculate reward for reward checkpoint.
func GetRewardForCheckpoint(chain consensus.ChainReader, blockSignerAddr common.Address, number uint64, rCheckpoint uint64, client bind.ContractBackend, totalSigner *uint64) (map[common.Address]*rewardLog, error) {
	// Not reward for singer of genesis block and only calculate reward at checkpoint block.
	startBlockNumber := number - (rCheckpoint * 2) + 1
	endBlockNumber := startBlockNumber + rCheckpoint - 1
	signers := make(map[common.Address]*rewardLog)

	for i := startBlockNumber; i <= endBlockNumber; i++ {
		block := chain.GetHeaderByNumber(i)
		addrs, err := GetSignersFromContract(blockSignerAddr, client, block.Hash())
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
func CalculateRewardForSigner(chainReward *big.Int, signers map[common.Address]*rewardLog, totalSigner uint64) (map[common.Address]*big.Int, error) {
	resultSigners := make(map[common.Address]*big.Int)
	// Add reward for signers.
	if totalSigner > 0 {
		for signer, rLog := range signers {
			// Add reward for signer.
			calcReward := new(big.Int)
			calcReward.Div(chainReward, new(big.Int).SetUint64(totalSigner))
			calcReward.Mul(calcReward, new(big.Int).SetUint64(rLog.Sign))
			rLog.Reward = calcReward

			resultSigners[signer] = calcReward
		}
	}
	jsonSigners, err := json.Marshal(signers)
	if err != nil {
		log.Error("Fail to parse json signers", "error", err)
		return nil, err
	}
	log.Info("Signers data", "signers", string(jsonSigners), "totalSigner", totalSigner, "totalReward", chainReward)

	return resultSigners, nil
}

// Get candidate owner by address.
func GetCandidatesOwnerBySigner(validator *contractValidator.TomoValidator, signerAddr common.Address) common.Address {
	owner := signerAddr
	opts := new(bind.CallOpts)
	owner, err := validator.GetCandidateOwner(opts, signerAddr)
	if err != nil {
		log.Error("Fail get candidate owner", "error", err)
		return owner
	}

	return owner
}

// Calculate reward for holders.
func CalculateRewardForHolders(validator *contractValidator.TomoValidator, state *state.StateDB, signer common.Address, calcReward *big.Int) error {
	rewards, err := GetRewardBalancesRate(signer, calcReward, validator)
	if err != nil {
		return err
	}
	if len(rewards) > 0 {
		for holder, reward := range rewards {
			state.AddBalance(holder, reward)
		}
	}
	return nil
}

// Get reward balance rates for master node, founder and holders.
func GetRewardBalancesRate(masterAddr common.Address, totalReward *big.Int, validator *contractValidator.TomoValidator) (map[common.Address]*big.Int, error) {
	owner := GetCandidatesOwnerBySigner(validator, masterAddr)
	balances := make(map[common.Address]*big.Int)
	rewardMaster := new(big.Int).Mul(totalReward, new(big.Int).SetInt64(RewardMasterPercent))
	rewardMaster = new(big.Int).Div(rewardMaster, new(big.Int).SetInt64(100))
	balances[owner] = rewardMaster
	// Get voters for masternode.
	opts := new(bind.CallOpts)
	voters, err := validator.GetVoters(opts, masterAddr)
	if err != nil {
		log.Error("Fail to get voters", "error", err)
		return nil, err
	}

	if len(voters) > 0 {
		totalVoterReward := new(big.Int).Mul(totalReward, new(big.Int).SetUint64(RewardVoterPercent))
		totalVoterReward = new(big.Int).Div(totalVoterReward, new(big.Int).SetUint64(100))
		totalCap := new(big.Int)
		// Get voters capacities.
		voterCaps := make(map[common.Address]*big.Int)
		for _, voteAddr := range voters {
			voterCap, err := validator.GetVoterCap(opts, masterAddr, voteAddr)
			if err != nil {
				log.Error("Fail to get vote capacity", "error", err)
				return nil, err
			}

			totalCap.Add(totalCap, voterCap)
			voterCaps[voteAddr] = voterCap
		}
		if totalCap.Cmp(new(big.Int).SetInt64(0)) > 0 {
			for addr, voteCap := range voterCaps {
				// Only valid voter has cap > 0.
				if voteCap.Cmp(new(big.Int).SetInt64(0)) > 0 {
					rcap := new(big.Int).Mul(totalVoterReward, voteCap)
					rcap = new(big.Int).Div(rcap, totalCap)
					if balances[addr] != nil {
						balances[addr].Add(balances[addr], rcap)
					} else {
						balances[addr] = rcap
					}
				}
			}
		}
	}

	foudationReward := new(big.Int).Mul(totalReward, new(big.Int).SetInt64(RewardFoundationPercent))
	foudationReward = new(big.Int).Div(foudationReward, new(big.Int).SetInt64(100))
	balances[common.HexToAddress(FoudationWalletAddr)] = foudationReward

	jsonHolders, err := json.Marshal(balances)
	if err != nil {
		log.Error("Fail to parse json holders", "error", err)
		return nil, err
	}
	log.Info("Holders reward", "holders", string(jsonHolders), "master node", masterAddr.String())

	return balances, nil
}
