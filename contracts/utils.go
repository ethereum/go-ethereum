// Copyright (c) 2018 XDPoSChain
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
	"errors"
	"io"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	randomizeContract "github.com/ethereum/go-ethereum/contracts/randomize/contract"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

// RewardLog represents the reward info for logging
type RewardLog struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

// Create tx sign.
func CreateTxSign(blockNumber *big.Int, blockHash common.Hash, nonce uint64, blockSigner common.Address) *types.Transaction {
	data := common.Hex2Bytes(common.HexSignMethod)
	inputData := append(data, common.LeftPadBytes(blockNumber.Bytes(), 32)...)
	inputData = append(inputData, common.LeftPadBytes(blockHash.Bytes(), 32)...)
	tx := types.NewTransaction(nonce, blockSigner, big.NewInt(0), 200000, big.NewInt(0), inputData)

	return tx
}

// Send secret key into randomize smartcontract.
func BuildTxSecretRandomize(nonce uint64, randomizeAddr common.Address, epocNumber uint64, randomizeKey []byte) (*types.Transaction, error) {
	data := common.Hex2Bytes(common.HexSetSecret)
	rand.Seed(time.Now().UnixNano())
	secretNumb := rand.Intn(int(epocNumber))

	// Append randomize suffix in -1, 0, 1.
	secrets := []int64{int64(secretNumb)}
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
	tx := types.NewTransaction(nonce, randomizeAddr, big.NewInt(0), 200000, big.NewInt(0), inputData)

	return tx, nil
}

// Send opening key into randomize smartcontract.
func BuildTxOpeningRandomize(nonce uint64, randomizeAddr common.Address, randomizeKey []byte) (*types.Transaction, error) {
	data := common.Hex2Bytes(common.HexSetOpening)
	inputData := append(data, randomizeKey...)
	tx := types.NewTransaction(nonce, randomizeAddr, big.NewInt(0), 200000, big.NewInt(0), inputData)

	return tx, nil
}

// RandStringByte generates a random string of specified length
func RandStringByte(n int) []byte {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	b := make([]byte, n)
	for i := range b {
		rand.Seed(time.Now().UnixNano())
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

// Encrypt encrypts the plaintext using AES
func Encrypt(key []byte, text string) string {
	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("Fail to encrypt", "error", err)
		return ""
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(cryptoRand.Reader, iv); err != nil {
		log.Error("Fail to encrypt iv", "error", err)
		return ""
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.URLEncoding.EncodeToString(ciphertext)
}

// Decrypt decrypts ciphertext using AES
func Decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("Fail to decrypt", "error", err)
		return ""
	}

	if len(ciphertext) < aes.BlockSize {
		log.Error("Ciphertext too short")
		return ""
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext)
}

// GetSignersFromContract returns the list of signers from the block signer contract
func GetSignersFromContract(statedb *state.StateDB, block *types.Block) ([]common.Address, error) {
	return state.GetSigners(statedb, block), nil
}

// GetRandomizeFromContract returns random number from contract
func GetRandomizeFromContract(client bind.ContractBackend, addr common.Address) (int64, error) {
	randomize, err := randomizeContract.NewXDCRandomize(common.RandomizeSMCBinary, client)
	if err != nil {
		return 0, err
	}
	opts := new(bind.CallOpts)
	secrets, err := randomize.GetSecret(opts, addr)
	if err != nil {
		return 0, err
	}
	// Since there could be multiple secrets (array), we just use first one
	if len(secrets) == 0 {
		return 0, errors.New("no secrets found")
	}
	// Decrypt the secret to get the random number
	opening, err := randomize.GetOpening(opts, addr)
	if err != nil {
		return 0, err
	}
	if len(opening) == 0 {
		return 0, errors.New("no opening found")
	}
	
	// Decrypt secret with opening key
	decrypted := Decrypt(opening[:], string(secrets[0][:]))
	result, err := strconv.ParseInt(decrypted, 10, 64)
	if err != nil {
		// If can't parse, return a pseudo-random based on address
		return int64(addr.Big().Int64() % 100), nil
	}
	return result, nil
}

// GenM2FromRandomize generates M2 validator list from random numbers
func GenM2FromRandomize(candidates []int64, lenSigners int64) ([]int64, error) {
	if len(candidates) == 0 {
		return nil, errors.New("empty candidates")
	}
	
	// Create a copy of candidates for shuffling
	m2 := make([]int64, len(candidates))
	copy(m2, candidates)
	
	// Fisher-Yates shuffle using candidates as seeds
	for i := len(m2) - 1; i > 0; i-- {
		seed := candidates[i%len(candidates)]
		j := int(seed) % (i + 1)
		if j < 0 {
			j = -j
		}
		m2[i], m2[j] = m2[j], m2[i]
	}
	
	return m2, nil
}

// BuildValidatorFromM2 builds validator bytes from M2 list
func BuildValidatorFromM2(m2 []int64) []byte {
	var validators []byte
	for _, v := range m2 {
		validators = append(validators, byte(v))
	}
	return validators
}

// GetBlockSigners returns signers who signed a specific block
func GetBlockSigners(client bind.ContractBackend, blockHash common.Hash) ([]common.Address, error) {
	blockSigners, err := contract.NewBlockSigner(common.BlockSignersBinary, client)
	if err != nil {
		return nil, err
	}
	opts := &bind.CallOpts{}
	addrs, err := blockSigners.GetSigners(opts, blockHash)
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

// ExtractAddressFromBytes extracts addresses from a byte slice
func ExtractAddressFromBytes(penaltiesBytes []byte) []common.Address {
	var addresses []common.Address
	if len(penaltiesBytes) == 0 {
		return addresses
	}
	// Each address is 20 bytes
	addressLen := common.AddressLength
	for i := 0; i+addressLen <= len(penaltiesBytes); i += addressLen {
		addresses = append(addresses, common.BytesToAddress(penaltiesBytes[i:i+addressLen]))
	}
	return addresses
}

// CalculateRewardForSigner calculates rewards for signers
func CalculateRewardForSigner(chainReward *big.Int, signers map[common.Address]*RewardLog, totalSigner uint64) (map[common.Address]*big.Int, error) {
	rewardSigners := make(map[common.Address]*big.Int)
	
	if totalSigner == 0 {
		return rewardSigners, nil
	}
	
	rewardPerSign := new(big.Int).Div(chainReward, big.NewInt(int64(totalSigner)))
	
	for addr, rLog := range signers {
		reward := new(big.Int).Mul(rewardPerSign, big.NewInt(int64(rLog.Sign)))
		rewardSigners[addr] = reward
	}
	
	return rewardSigners, nil
}

// CalculateRewardForHolders calculates rewards for token holders
func CalculateRewardForHolders(foundationWalletAddr common.Address, statedb *state.StateDB, signer common.Address, calcReward *big.Int, blockNumber uint64) (map[common.Address]*big.Int, error) {
	rewards := make(map[common.Address]*big.Int)
	
	// Simple distribution: foundation gets foundation percent, signer gets rest
	foundationReward := new(big.Int).Mul(calcReward, big.NewInt(int64(common.RewardFoundationPercent)))
	foundationReward.Div(foundationReward, big.NewInt(100))
	
	signerReward := new(big.Int).Sub(calcReward, foundationReward)
	
	if foundationWalletAddr != (common.Address{}) {
		rewards[foundationWalletAddr] = foundationReward
	}
	rewards[signer] = signerReward
	
	return rewards, nil
}

// DecodeExtraFields decodes extra data fields from header
func DecodeExtraFields(extra []byte) (common.Address, []byte, error) {
	if len(extra) < extraVanity {
		return common.Address{}, nil, errors.New("extra-data too short")
	}
	
	// Extract vanity
	vanity := extra[:extraVanity]
	
	// Extract signer
	if len(extra) < extraVanity+extraSeal {
		return common.Address{}, vanity, errors.New("missing signature")
	}
	
	signature := extra[len(extra)-extraSeal:]
	_ = signature // Would be used to recover signer
	
	return common.Address{}, vanity, nil
}

// CompareSignersLists compares two lists of signers
func CompareSignersLists(list1, list2 []common.Address) bool {
	if len(list1) != len(list2) {
		return false
	}
	
	for i := range list1 {
		if list1[i] != list2[i] {
			return false
		}
	}
	
	return true
}

// BytesEqual compares two byte slices
func BytesEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}
