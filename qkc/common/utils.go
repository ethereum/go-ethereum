// Ported verbatim from github.com/QuarkChain/goquarkchain/common (byte-compatible).

package common

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"math"
	"math/big"
	"math/bits"
	"net"
	"reflect"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	DirectionToGenesis = uint8(0)
	DirectionToTip     = uint8(1)

	SkipHash   = uint8(0)
	SkipHeight = uint8(1)
)

var (
	EmptyHash  = ethCommon.Hash{}
	uint128Max = GetUint128Max()
)

func GetUint128Max() *big.Int {
	pow2_64 := new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), ethCommon.Big1)
	pow2_128 := new(big.Int).Mul(pow2_64, pow2_64)
	return new(big.Int).Sub(pow2_128, ethCommon.Big1)
}

func BiggerThanUint128Max(data *big.Int) bool {
	return data.Cmp(uint128Max) > 0
}

/*
0b101, 0b11 -> True
0b101, 0b10 -> False
*/
func MasksHaveOverlap(m1, m2 uint32) bool {
	i1 := IntLeftMostBit(m1)
	i2 := IntLeftMostBit(m2)
	if i1 > i2 {
		i1 = i2
	}
	bitMask := uint32((1 << (i1 - 1)) - 1)
	return (m1 & bitMask) == (m2 & bitMask)
}

// IsP2 is check num is 2^x
func IsP2(shardSize uint32) bool {
	return (shardSize & (shardSize - 1)) == 0
}

// IntLeftMostBit left most bit
func IntLeftMostBit(v uint32) uint32 {
	return uint32(32 - bits.LeadingZeros32(v))
}

func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func GetIPV4Addr() (string, error) {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1", err
	}

	for _, addr := range addrs {
		ipNet, isIpNet := addr.(*net.IPNet)
		if isIpNet && !ipNet.IP.IsLoopback() {
			ipv4 := ipNet.IP.To4()
			if ipv4 != nil {
				return ipv4.String(), nil
			}
		}
	}
	log.Error("ipv4 addr not found", "addr", addrs)
	return "127.0.0.1", nil
}

func IsLocalIP(ip string) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for i := range addrs {
		intf, _, err := net.ParseCIDR(addrs[i].String())
		if err != nil {
			return false
		}
		if net.ParseIP(ip).Equal(intf) {
			return true
		}
	}
	return false
}

func IsNil(data interface{}) bool {
	return data == nil || reflect.ValueOf(data).IsNil()
}

// ConstMinorBlockRewardCalculator blockReward struct
type ConstMinorBlockRewardCalculator struct {
}

// GetBlockReward getBlockReward
func (c *ConstMinorBlockRewardCalculator) GetBlockReward() *big.Int {
	data := new(big.Int).SetInt64(100)
	return new(big.Int).Mul(data, new(big.Int).SetInt64(1000000000000000000))
}

func BigIntMulBigRat(bigInt *big.Int, bigRat *big.Rat) *big.Int {
	bigRat1 := new(big.Rat).Set(bigRat)
	ans := new(big.Int).Mul(bigInt, bigRat1.Num())
	ans.Div(ans, bigRat1.Denom())
	return ans
}

// Uint32ToBytes trans uint32 num to bytes
func Uint32ToBytes(n uint32) []byte {
	Bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(Bytes, n)
	return Bytes
}

func Uint64ToBytes(n uint64) []byte {
	Bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(Bytes, n)
	return Bytes
}

func BytesToUint32(byte []byte) uint32 {
	bytesBuffer := bytes.NewBuffer(byte)
	var x uint32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}

func EncodeToByte32(data uint64) []byte {
	ret := make([]byte, 32)
	binary.BigEndian.PutUint64(ret[24:], data)
	return ret
}

func BigToByte32(data *big.Int) []byte {
	dataBytes := data.Bytes()
	lenData := len(dataBytes)
	if lenData > 32 {
		panic("data's len should <= 32")
	}
	ret := make([]byte, 32)
	copy(ret[(32-lenData):], dataBytes)
	return ret
}

func Has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func RemoveDuplicate(data []uint64) []uint64 {
	newData := make([]uint64, 0, len(data))
	for _, iData := range data {
		if len(newData) == 0 {
			newData = append(newData, iData)
		} else {
			for k, v := range newData {
				if v == iData {
					break
				}
				if k == len(newData)-1 {
					newData = append(newData, iData)
				}
			}
		}
	}
	return newData
}
