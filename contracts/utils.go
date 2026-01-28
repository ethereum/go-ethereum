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
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/XDPoS/utils"
	randomizeContract "github.com/ethereum/go-ethereum/contracts/randomize/contract"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

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

// Send opening to randomize SMC.
func BuildTxOpeningRandomize(nonce uint64, randomizeAddr common.Address, randomizeKey []byte) (*types.Transaction, error) {
	data := common.Hex2Bytes(common.HexSetOpening)
	inputData := append(data, randomizeKey...)
	tx := types.NewTransaction(nonce, randomizeAddr, big.NewInt(0), 200000, big.NewInt(0), inputData)

	return tx, nil
}

// Get random from randomize contract.
func GetRandomizeFromContract(client bind.ContractBackend, addrMasternode common.Address) (int64, error) {
	randomize, err := randomizeContract.NewXDCRandomize(common.RandomizeSMCBinary, client)
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
func GenM2FromRandomize(randomizes []int64, lenSigners int64) ([]int64, error) {
	blockValidator := NewSlice(int64(0), lenSigners, 1)
	randIndexs := make([]int64, lenSigners)
	total := int64(0)
	var temp int64 = 0
	for _, j := range randomizes {
		total += j
	}
	rand.Seed(total)
	for i := len(blockValidator) - 1; i >= 0; i-- {
		blockLength := len(blockValidator) - 1
		if blockLength <= 1 {
			blockLength = 1
		}
		randomIndex := int64(rand.Intn(blockLength))
		temp = blockValidator[randomIndex]
		blockValidator[randomIndex] = blockValidator[i]
		blockValidator[i] = temp
		blockValidator = append(blockValidator[:i], blockValidator[i+1:]...)
		randIndexs[i] = temp
	}

	return randIndexs, nil
}

// Get validators from m2 array integer.
func BuildValidatorFromM2(listM2 []int64) []byte {
	var validatorBytes []byte
	for _, numberM2 := range listM2 {
		// Convert number to byte.
		m2Byte := common.LeftPadBytes([]byte(fmt.Sprintf("%d", numberM2)), utils.M2ByteLength)
		validatorBytes = append(validatorBytes, m2Byte...)
	}

	return validatorBytes
}

// Decode validator hex string.
func DecodeValidatorsHexData(validatorsStr string) ([]int64, error) {
	validatorsByte, err := hexutil.Decode(validatorsStr)
	if err != nil {
		return nil, err
	}

	return utils.ExtractValidatorsFromBytes(validatorsByte), nil
}

// Decrypt randomize from secrets and opening.
func DecryptRandomizeFromSecretsAndOpening(secrets [][32]byte, opening [32]byte) (int64, error) {
	var random int64
	if len(secrets) > 0 {
		for _, secret := range secrets {
			trimSecret := bytes.TrimLeft(secret[:], "\x00")
			decryptSecret := Decrypt(opening[:], string(trimSecret))
			if isInt(decryptSecret) {
				intNumber, err := strconv.Atoi(decryptSecret)
				if err != nil {
					log.Error("Can not convert string to integer", "error", err)
					return -1, err
				}
				random = int64(intNumber)
			}
		}
	}

	return random, nil
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
	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("Fail to encrypt", "err", err)
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(cryptoRand.Reader, iv); err != nil {
		log.Error("Fail to encrypt iv", "err", err)
		return ""
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
		log.Error("Fail to decrypt", "err", err)
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		log.Error("ciphertext too short")
		return ""
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext[:])
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
