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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	randomizeContract "github.com/ethereum/go-ethereum/contracts/randomize/contract"
	contractValidator "github.com/ethereum/go-ethereum/contracts/validator/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"io"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

const (
	HexSignMethod           = "e341eaa4"
	RewardMasterPercent     = 40
	RewardVoterPercent      = 50
	RewardFoundationPercent = 10
	HexSetSecret            = "34d38600"
	HexSetOpening           = "e11f5ba2"
	EpocBlockSecret         = 950
	EpocBlockOpening        = 980
	EpocBlockRandomize      = 900
	M2ByteLength            = 4
	extraVanity             = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal               = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

type rewardLog struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

// Send tx sign for block number to smart contract blockSigner.
func CreateTransactionSign(chainConfig *params.ChainConfig, pool *core.TxPool, manager *accounts.Manager, block *types.Block, chainDb ethdb.Database) error {
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
		err = pool.AddLocal(txSigned)
		if err != nil {
			log.Error("Fail to add tx sign to local pool.", "error", err)
		}

		// Create secret tx.
		blockNumber := block.Number().Uint64()
		checkNumber := blockNumber % chainConfig.Posv.Epoch
		// Generate random private key and save into chaindb.
		randomizeKeyName := []byte("randomizeKey")
		exist, _ := chainDb.Has(randomizeKeyName)

		// Set secret for randomize.
		if !exist && checkNumber > 0 && EpocBlockSecret <= checkNumber && EpocBlockOpening > checkNumber {
			// Only process when private key empty in state db.
			// Save randomize key into state db.
			randomizeKeyValue := RandStringByte(32)
			chainDb.Put(randomizeKeyName, randomizeKeyValue)

			tx, err := BuildTxSecretRandomize(nonce, common.HexToAddress(common.RandomizeSMC), chainConfig.Posv.Epoch, randomizeKeyValue)
			if err != nil {
				log.Error("Fail to get tx opening for randomize", "error", err)
				return err
			}
			txSigned, err := wallet.SignTx(account, tx, chainConfig.ChainId)
			if err != nil {
				log.Error("Fail to create tx secret", "error", err)
				return err
			}
			// Add tx signed to local tx pool.
			err = pool.AddLocal(txSigned)
			if err != nil {
				log.Error("Fail to add tx secret to local pool.", "error", err)
			}
		}

		// Set opening for randomize.
		if exist && checkNumber > 0 && EpocBlockOpening <= checkNumber && EpocBlockRandomize >= checkNumber {
			randomizeKeyValue, err := chainDb.Get(randomizeKeyName)
			if err != nil {
				log.Error("Fail to get randomize key from state db.", "error", err)
			}

			tx, err := BuildTxOpeningRandomize(nonce, common.HexToAddress(common.RandomizeSMC), randomizeKeyValue)
			if err != nil {
				log.Error("Fail to get tx opening for randomize", "error", err)
				return err
			}
			txSigned, err := wallet.SignTx(account, tx, chainConfig.ChainId)
			if err != nil {
				log.Error("Fail to create tx opening", "error", err)
				return err
			}
			// Add tx to pool.
			err = pool.AddLocal(txSigned)
			if err != nil {
				log.Error("Fail to add tx opening to local pool.", "error", err)
			}

			// Clear randomize key in state db.
			chainDb.Delete(randomizeKeyName)
		}
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

// Send secret key into randomize smartcontract.
func BuildTxSecretRandomize(nonce uint64, randomizeAddr common.Address, epocNumber uint64, randomizeKey []byte) (*types.Transaction, error) {
	data := common.Hex2Bytes(HexSetSecret)
	secretMax := math.Round(float64(epocNumber / 10))
	secrets := Shuffle(NewSlice(0, int64(secretMax), 1))

	// Append randomize suffix in -1, 0, 1.
	arrSuffix := []int64{-1, 1}
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(arrSuffix))
	secrets = append(secrets, arrSuffix[randIndex])

	sizeOfArray := int64(32)
	// Build extra data for tx with first position is size of array byte and second position are length of array byte.
	arrSizeOfSecrets := common.LeftPadBytes(new(big.Int).SetInt64(sizeOfArray).Bytes(), 32)
	arrLengthOfSecrets := common.LeftPadBytes(new(big.Int).SetInt64(int64(len(secrets))).Bytes(), 32)
	inputData := append(data, arrSizeOfSecrets...)
	inputData = append(inputData, arrLengthOfSecrets...)
	for _, secret := range secrets {
		encryptSecret := Encrypt(randomizeKey, new(big.Int).SetInt64(secret).String())
		inputData = append(inputData, common.LeftPadBytes([]byte(encryptSecret), int(sizeOfArray))...)
	}
	tx := types.NewTransaction(nonce, randomizeAddr, big.NewInt(0), 4200000, big.NewInt(0), inputData)

	return tx, nil
}

// Send opening to randomize SMC.
func BuildTxOpeningRandomize(nonce uint64, randomizeAddr common.Address, randomizeKey []byte) (*types.Transaction, error) {
	data := common.Hex2Bytes(HexSetOpening)
	inputData := append(data, randomizeKey...)
	tx := types.NewTransaction(nonce, randomizeAddr, big.NewInt(0), 4200000, big.NewInt(0), inputData)

	return tx, nil
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

// Get random from randomize contract.
func GetRandomizeFromContract(client bind.ContractBackend, addrMasternode common.Address) ([]int64, error) {
	randomize, err := randomizeContract.NewTomoRandomize(common.HexToAddress(common.RandomizeSMC), client)
	if err != nil {
		log.Error("Fail to get instance of randomize", "error", err)
	}
	opts := new(bind.CallOpts)
	secrets, err := randomize.GetSecret(opts, addrMasternode)
	if err != nil {
		log.Error("Fail get secrets from randomize", "error", err)
	}
	opening, err := randomize.GetOpening(opts, addrMasternode)
	if err != nil {
		log.Error("Fail get opening from randomize", "error", err)
	}

	return DecryptRandomizeFromSecretsAndOpening(secrets, opening)
}

// Generate m2 listing from randomize array.
func GenM2FromRandomize(randomizes [][]int64) ([]int64, error) {
	// Separate array.
	arrRandomizes := TransposeMatrix(randomizes)
	lenRandomize := len(arrRandomizes)
	arrayA := arrRandomizes[:lenRandomize-2]
	arrayB := arrRandomizes[lenRandomize-1:]

	matrixResult, err := DotMatrix(arrayA, arrayB)
	if err != nil {
		log.Error("Fail to dot matrix", "error", err)

		return nil, err
	}
	lenMasternode := len(arrayB[0])
	result := make([]int64, lenRandomize)
	for i, v := range matrixResult {
		result[i] = int64(math.Abs(float64(v))) % int64(lenMasternode)
	}

	return result, nil
}

// Get validators from m2 array integer.
func BuildValidatorFromM2(listM2 []int64) []byte {
	var validatorBytes []byte
	for _, numberM2 := range listM2 {
		// Convert number to byte.
		m2Byte := common.LeftPadBytes([]byte(fmt.Sprintf("%d", numberM2)), M2ByteLength)
		validatorBytes = append(validatorBytes, m2Byte...)
	}

	return validatorBytes
}

// Extract validators from byte array.
func ExtractValidatorsFromBytes(byteValidators []byte) []int64 {
	lenValidator := len(byteValidators) / M2ByteLength
	var validators []int64
	for i := 0; i < lenValidator; i++ {
		trimByte := bytes.Trim(byteValidators[i*M2ByteLength:(i+1)*M2ByteLength], "\x00")
		intNumber, err := strconv.Atoi(string(trimByte))
		if err != nil {
			log.Error("Can not convert string to integer", "error", err)
		}
		validators = append(validators, int64(intNumber))
	}

	return validators
}

// Decode validator hex string.
func DecodeValidatorsHexData(validatorsStr string) ([]int64, error) {
	validatorsByte, err := hexutil.Decode(validatorsStr)
	if err != nil {
		return nil, err
	}

	return ExtractValidatorsFromBytes(validatorsByte), nil
}

// Decrypt randomize from secrets and opening.
func DecryptRandomizeFromSecretsAndOpening(secrets [][32]byte, opening [32]byte) ([]int64, error) {
	var random []int64
	var completedRandomize []int64
	random = make([]int64, len(secrets))
	if len(secrets) > 0 {
		for i, secret := range secrets {
			trimSecret := bytes.TrimLeft(secret[:], "\x00")
			decryptSecret := Decrypt(opening[:], string(trimSecret))
			if isInt(decryptSecret) {
				intNumber, err := strconv.Atoi(decryptSecret)
				if err != nil {
					log.Error("Can not convert string to integer", "error", err)
				}
				random[i] = int64(intNumber)
			}
		}
	}

	// Generate full randomize.
	randomMax := len(random) - 1
	if randomMax > 0 {
		for i, randNumber := range random {
			if i < randomMax {
				for j := 0; j < 10; j++ {
					completedRandomize = append(completedRandomize, randNumber*10+int64(j))
				}
			}
		}
		completedRandomize = append(completedRandomize, random[randomMax])
	}
	log.Error("random", "completedRandomize", completedRandomize, "randomMax", randomMax)

	return completedRandomize, nil
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
func CalculateRewardForHolders(foudationWalletAddr common.Address, validator *contractValidator.TomoValidator, state *state.StateDB, signer common.Address, calcReward *big.Int) error {
	rewards, err := GetRewardBalancesRate(foudationWalletAddr, signer, calcReward, validator)
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
func GetRewardBalancesRate(foudationWalletAddr common.Address, masterAddr common.Address, totalReward *big.Int, validator *contractValidator.TomoValidator) (map[common.Address]*big.Int, error) {
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
	balances[foudationWalletAddr] = foudationReward

	jsonHolders, err := json.Marshal(balances)
	if err != nil {
		log.Error("Fail to parse json holders", "error", err)
		return nil, err
	}
	log.Info("Holders reward", "holders", string(jsonHolders), "master node", masterAddr.String())

	return balances, nil
}

// Dynamic generate array sequence of numbers.
func NewSlice(start int64, end int64, step int64) []int64 {
	s := make([]int64, end-start)
	for i := range s {
		s[i] = start
		start += step
	}

	return s
}

// Shuffle array.
func Shuffle(slice []int64) []int64 {
	newSlice := make([]int64, len(slice))
	copy(newSlice, slice)

	for i := 0; i < len(slice)-1; i++ {
		rand.Seed(time.Now().UnixNano())
		randIndex := rand.Intn(len(newSlice))
		x := newSlice[i]
		newSlice[i] = newSlice[randIndex]
		newSlice[randIndex] = x
	}

	return newSlice
}

// encrypt string to base64 crypto using AES
func Encrypt(key []byte, text string) string {
	// key := []byte(keyText)
	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(cryptoRand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

// decrypt from base64 to decrypted string
func Decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}

// Generate random string.
func RandStringByte(n int) []byte {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	b := make([]byte, n)
	for i := range b {
		rand.Seed(time.Now().UnixNano())
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

// Helper function check string is numeric.
func isInt(strNumber string) bool {
	if _, err := strconv.Atoi(strNumber); err == nil {
		return true
	} else {
		return false
	}
}

// Helper function for transpose matrix.
func TransposeMatrix(x [][]int64) [][]int64 {
	out := make([][]int64, len(x[0]))
	for i := 0; i < len(x); i += 1 {
		for j := 0; j < len(x[0]); j += 1 {
			out[j] = append(out[j], x[i][j])
		}
	}
	return out
}

// Helper function for multiplication matrix.
func DotMatrix(x, y [][]int64) ([]int64, error) {
	if len(x[0]) != len(y[0]) {
		return nil, errors.New("Can't do matrix multiplication.")
	}

	out := make([]int64, len(x))
	for i := 0; i < len(x); i += 1 {
		for j := 0; j < len(y[0]); j += 1 {
			out[i] += x[i][j] * y[0][j]
		}
	}

	return out, nil
}

// Get masternodes address from checkpoint Header.
func GetMasternodesFromCheckpointHeader(checkpointHeader *types.Header) []common.Address {
	masternodes := make([]common.Address, (len(checkpointHeader.Extra)-extraVanity-extraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], checkpointHeader.Extra[extraVanity+i*common.AddressLength:])
	}
	return masternodes
}

// Get m2 list from checkpoint block.
func GetM2FromCheckpointBlock(checkpointBlock types.Block) ([]common.Address, error) {
	if checkpointBlock.Number().Int64()%EpocBlockRandomize != 0 {
		return nil, errors.New("This block is not checkpoint block epoc.")
	}

	// Get singers from this block.
	masternodes := GetMasternodesFromCheckpointHeader(checkpointBlock.Header())
	validators := ExtractValidatorsFromBytes(checkpointBlock.Header().Validators)

	var m2List []common.Address
	for validatorIndex := range validators {
		m2List = append(m2List, masternodes[validatorIndex])
	}

	return m2List, nil
}
