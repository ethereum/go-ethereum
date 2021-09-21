package state

import (
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto"
)

func GetLocSimpleVariable(slot uint64) common.Hash {
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	return slotHash
}

func GetLocMappingAtKey(key common.Hash, slot uint64) *big.Int {
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	retByte := crypto.Keccak256(key.Bytes(), slotHash.Bytes())
	ret := new(big.Int)
	ret.SetBytes(retByte)
	return ret
}

func GetLocDynamicArrAtElement(slotHash common.Hash, index uint64, elementSize uint64) common.Hash {
	slotKecBig := crypto.Keccak256Hash(slotHash.Bytes()).Big()
	//arrBig = slotKecBig + index * elementSize
	arrBig := slotKecBig.Add(slotKecBig, new(big.Int).SetUint64(index*elementSize))
	return common.BigToHash(arrBig)
}

func GetLocFixedArrAtElement(slot uint64, index uint64, elementSize uint64) common.Hash {
	slotBig := new(big.Int).SetUint64(slot)
	arrBig := slotBig.Add(slotBig, new(big.Int).SetUint64(index*elementSize))
	return common.BigToHash(arrBig)
}

func GetLocOfStructElement(locOfStruct *big.Int, elementSlotInstruct *big.Int) common.Hash {
	return common.BigToHash(new(big.Int).Add(locOfStruct, elementSlotInstruct))
}
