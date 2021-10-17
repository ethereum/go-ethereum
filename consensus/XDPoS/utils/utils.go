package utils

import (
	"bytes"
	"errors"
	"reflect"
	"sort"
	"strconv"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

func Position(list []common.Address, x common.Address) int {
	for i, item := range list {
		if item == x {
			return i
		}
	}
	return -1
}

func Hop(len, pre, cur int) int {
	switch {
	case pre < cur:
		return cur - (pre + 1)
	case pre > cur:
		return (len - pre) + (cur - 1)
	default:
		return len - 1
	}
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
			return []int64{}
		}
		validators = append(validators, int64(intNumber))
	}

	return validators
}

// Get masternodes address from checkpoint Header.
func GetMasternodesFromCheckpointHeader(checkpointHeader *types.Header) []common.Address {
	masternodes := make([]common.Address, (len(checkpointHeader.Extra)-ExtraVanity-ExtraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], checkpointHeader.Extra[ExtraVanity+i*common.AddressLength:])
	}
	return masternodes
}

// Get m2 list from checkpoint block.
func GetM1M2FromCheckpointHeader(checkpointHeader *types.Header, currentHeader *types.Header, config *params.ChainConfig) (map[common.Address]common.Address, error) {
	if checkpointHeader.Number.Uint64()%common.EpocBlockRandomize != 0 {
		return nil, errors.New("This block is not checkpoint block epoc.")
	}
	// Get signers from this block.
	masternodes := GetMasternodesFromCheckpointHeader(checkpointHeader)
	validators := ExtractValidatorsFromBytes(checkpointHeader.Validators)
	m1m2, _, err := GetM1M2(masternodes, validators, currentHeader, config)
	if err != nil {
		return map[common.Address]common.Address{}, err
	}
	return m1m2, nil
}

func GetM1M2(masternodes []common.Address, validators []int64, currentHeader *types.Header, config *params.ChainConfig) (map[common.Address]common.Address, uint64, error) {
	m1m2 := map[common.Address]common.Address{}
	maxMNs := len(masternodes)
	moveM2 := uint64(0)
	if len(validators) < maxMNs {
		return nil, moveM2, errors.New("len(m2) is less than len(m1)")
	}
	if maxMNs > 0 {
		isForked := config.IsTIPRandomize(currentHeader.Number)
		if isForked {
			moveM2 = ((currentHeader.Number.Uint64() % config.XDPoS.Epoch) / uint64(maxMNs)) % uint64(maxMNs)
		}
		for i, m1 := range masternodes {
			m2Index := uint64(validators[i] % int64(maxMNs))
			m2Index = (m2Index + moveM2) % uint64(maxMNs)
			m1m2[m1] = masternodes[m2Index]
		}
	}
	return m1m2, moveM2, nil
}

// compare 2 signers lists
// return true if they are same elements, otherwise return false
func CompareSignersLists(list1 []common.Address, list2 []common.Address) bool {
	if len(list1) == 0 && len(list2) == 0 {
		return true
	}
	sort.Slice(list1, func(i, j int) bool {
		return list1[i].String() <= list1[j].String()
	})
	sort.Slice(list2, func(i, j int) bool {
		return list2[i].String() <= list2[j].String()
	})
	return reflect.DeepEqual(list1, list2)
}

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
//type SignerFn func(accounts.Account, []byte) ([]byte, error)

// sigHash returns the hash which is used as input for the delegated-proof-of-stake
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func SigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	err := rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})
	if err != nil {
		log.Debug("Fail to encode", err)
	}
	hasher.Sum(hash[:0])
	return hash
}
