package utils

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"golang.org/x/crypto/sha3"
)

func Position(list []common.Address, x common.Address) int {
	for i, item := range list {
		if item == x {
			return i
		}
	}
	return -1
}

func Hop(length, preIndex, curIndex int) int {
	switch {
	case preIndex < curIndex:
		return curIndex - (preIndex + 1)
	case preIndex > curIndex:
		return (length - preIndex) + (curIndex - 1)
	default:
		return length - 1
	}
}

// Extract validators from byte array.
func ExtractValidatorsFromBytes(byteValidators []byte) ([]int64, error) {
	if len(byteValidators)%M2ByteLength != 0 {
		return []int64{}, fmt.Errorf("invalid byte array length %d for validators", len(byteValidators))
	}
	lenValidator := len(byteValidators) / M2ByteLength
	validators := make([]int64, 0, lenValidator)
	for i := range lenValidator {
		trimByte := bytes.Trim(byteValidators[i*M2ByteLength:(i+1)*M2ByteLength], "\x00")
		intNumber, err := strconv.ParseInt(string(trimByte), 10, 64)
		if err != nil {
			log.Error("Can not convert string to integer", "error", err)
			return []int64{}, fmt.Errorf("can not convert string %s to integer: %v", string(trimByte), err)
		}
		validators = append(validators, intNumber)
	}

	return validators, nil
}

// compare 2 signers lists
// return true if they are same elements, otherwise return false
func CompareSignersLists(list1 []common.Address, list2 []common.Address) bool {
	if len(list1) != len(list2) {
		return false
	}
	if len(list1) == 0 {
		return true
	}

	l1 := slices.Clone(list1)
	l2 := slices.Clone(list2)

	slices.SortFunc(l1, func(a, b common.Address) int {
		return bytes.Compare(a[:], b[:])
	})
	slices.SortFunc(l2, func(a, b common.Address) int {
		return bytes.Compare(a[:], b[:])
	})

	return slices.Equal(l1, l2)
}

// Decode extra fields for consensus version >= 2 (XDPoS 2.0 and future versions)
func DecodeBytesExtraFields(b []byte, val interface{}) error {
	if len(b) == 0 {
		return errors.New("extra field is 0 length")
	}
	// Prevent payload attack, limit the size of extra field to 20k bytes. Normal Extrafield payload is less than 7k bytes.
	if len(b) > 20000 {
		return errors.New("extra field is too long")
	}

	switch b[0] {
	case 2:
		return rlp.DecodeBytes(b[1:], val)
	default:
		return fmt.Errorf("consensus version %d is not defined, or this block is v1 block", b[0])
	}
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	err := rlp.Encode(hw, x)
	if err != nil {
		log.Error("[rlpHash] Fail to hash item", "Error", err)
	}
	hw.Sum(h[:0])
	return h
}
