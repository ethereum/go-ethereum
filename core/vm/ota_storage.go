// Copyright 2019 usechain Foundation Ltd

package vm

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"math/rand"

	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrUnknown          = errors.New("unknown error")
	ErrInvalidOTAAddr   = errors.New("invalid OTA addrss")
	ErrInvalidOTAAX     = errors.New("invalid OTA AX")
	ErrOTAExistAlready  = errors.New("OTA exist already")
	ErrOTABalanceIsZero = errors.New("OTA balance is 0")
)

// OTABalance2ContractAddr convert ota balance to ota storage address
//
// 1 eth --> (bigint)1000000000000000000 --> "0x0000000000000000000001000000000000000000"
//
func OTABalance2ContractAddr(balance *big.Int) common.Address {
	if balance == nil {
		return common.Address{}
	}

	return common.HexToAddress(balance.String())
	//	return common.BigToAddress(balance)
}

// GetAXFromUseAddr retrieve ota AX from ota UseAddr
func GetAXFromUseAddr(otaUseAddr []byte) ([]byte, error) {
	if len(otaUseAddr) != common.UAddressLength {
		return nil, ErrInvalidOTAAddr
	}

	return otaUseAddr[1 : 1+common.HashLength], nil
}

// IsAXPointToUseAddr check whether AX point to otaUseAddr or not
func IsAXPointToUseAddr(AX []byte, otaUseAddr []byte) bool {
	findAX, err := GetAXFromUseAddr(otaUseAddr)
	if err != nil {
		return false
	}

	return bytes.Equal(findAX, AX)
}

// GetOtaBalanceFromAX retrieve ota balance from ota AX
func GetOtaBalanceFromAX(statedb StateDB, otaAX []byte) (*big.Int, error) {
	if statedb == nil {
		return nil, ErrUnknown
	}

	if len(otaAX) != common.HashLength {
		return nil, ErrInvalidOTAAX
	}

	balance := statedb.GetStateByteArray(otaBalanceStorageAddr, common.BytesToHash(otaAX))
	if len(balance) == 0 {
		return common.Big0, nil
	}

	return new(big.Int).SetBytes(balance), nil
}

// SetOtaBalanceToAX set ota balance as 'balance'. Overwrite if ota balance exist already.
func SetOtaBalanceToAX(statedb StateDB, otaAX []byte, balance *big.Int) error {
	if statedb == nil || balance == nil {
		return ErrUnknown
	}

	if len(otaAX) != common.HashLength {
		return ErrInvalidOTAAX
	}

	statedb.SetStateByteArray(otaBalanceStorageAddr, common.BytesToHash(otaAX), balance.Bytes())
	return nil
}

// ChechOTAExist checks the OTA exist or not.
//
// In order to avoid additional ota have conflict with existing,
// even if AX exist in balance storage already, will return true.
func CheckOTAAXExist(statedb StateDB, otaAX []byte) (exist bool, balance *big.Int, err error) {
	if statedb == nil {
		return false, nil, ErrUnknown
	}

	if len(otaAX) != common.HashLength {
		return false, nil, ErrInvalidOTAAX
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX[:common.HashLength])
	if err != nil {
		return false, nil, err
	}

	if balance.Cmp(common.Big0) == 0 {
		return false, nil, nil
	}

	return true, balance, nil
}

func CheckOTALongAddrExist(statedb StateDB, otaLongAddr []byte) (exist bool, balance *big.Int, err error) {
	if statedb == nil {
		return false, nil, ErrUnknown
	}

	if len(otaLongAddr) != 33 {
		return false, nil, ErrInvalidOTAAX
	}

	otaAX := otaLongAddr[1 : 1+common.HashLength]
	if err != nil {
		return false, nil, err
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		return false, nil, err
	}

	if balance.Cmp(common.Big0) == 0 {
		return false, nil, nil
	}

	mptAddr := OTABalance2ContractAddr(balance)
	otaStored := statedb.GetStateByteArray(mptAddr, common.BytesToHash(otaAX))

	fmt.Println(common.ToHex(mptAddr.Bytes()))
	fmt.Println(common.ToHex(otaLongAddr))

	if otaStored == nil {
		return false, nil, nil
	}

	if !bytes.Equal(otaLongAddr, otaStored[:33]) {
		return false, nil, nil
	}

	return true, balance, nil
}

func BatCheckOTAExist(statedb StateDB, otaLongAddrs [][]byte) (exist bool, balance *big.Int, unexistOta []byte, err error) {
	if statedb == nil || len(otaLongAddrs) == 0 {
		return false, nil, nil, ErrUnknown
	}

	for _, otaLongAddr := range otaLongAddrs {
		if len(otaLongAddr) != 33 {
			return false, nil, otaLongAddr, ErrInvalidOTAAX
		}

		exist, balanceTmp, err := CheckOTALongAddrExist(statedb, otaLongAddr)
		if err != nil {
			return false, nil, otaLongAddr, err
		} else if !exist {
			return false, nil, otaLongAddr, errors.New("ota doesn't exist:" + common.ToHex(otaLongAddr))
		} else if balanceTmp.Cmp(common.Big0) == 0 {
			return false, nil, otaLongAddr, errors.New("ota balance is 0! ota:" + common.ToHex(otaLongAddr))
		} else if balance == nil {
			balance = balanceTmp
			continue
		} else if balance.Cmp(balanceTmp) != 0 {
			return false, nil, otaLongAddr, errors.New("otas have different balances! ota:" + common.ToHex(otaLongAddr))
		}
	}

	return true, balance, nil, nil
}

func GetUnspendOTATotalBalance(statedb StateDB) (*big.Int, error) {
	if statedb == nil {
		return nil, ErrUnknown
	}

	totalOTABalance, totalSpendedOTABalance := big.NewInt(0), big.NewInt(0)

	// total history OTA balance (include spended)
	statedb.ForEachStorageByteArray(otaBalanceStorageAddr, func(key common.Hash, value []byte) bool {
		if len(value) == 0 {
			log.Warn("total ota balance. value is empoty!", "key", key.String())
			return true
		}

		balance := new(big.Int).SetBytes(value)
		totalOTABalance.Add(totalOTABalance, balance)
		log.Debug("total ota balance.", "key", key.String(), "balance:", balance.String())
		return true
	})

	// total spended OTA balance
	statedb.ForEachStorageByteArray(otaImageStorageAddr, func(key common.Hash, value []byte) bool {
		if len(value) == 0 {
			log.Warn("total spended ota balance. value is empoty!", "key", key.String())
			return true
		}

		balance := new(big.Int).SetBytes(value)
		totalSpendedOTABalance.Add(totalSpendedOTABalance, balance)
		log.Debug("total spended ota balance.", "key", key.String(), "balance:", balance.String())
		return true
	})

	log.Debug("total unspended OTA balance", "total history OTA balance:", totalOTABalance.String(), "total spended OTA balance:", totalSpendedOTABalance.String())

	return totalOTABalance.Sub(totalOTABalance, totalSpendedOTABalance), nil
}

// setOTA storage ota info, include balance and UseAddr. Overwrite if ota exist already.
func setOTA(statedb StateDB, balance *big.Int, otaUseAddr []byte) error {
	if statedb == nil || balance == nil {
		return ErrUnknown
	}
	if len(otaUseAddr) != common.UAddressLength {
		return ErrInvalidOTAAddr
	}

	otaAX, _ := GetAXFromUseAddr(otaUseAddr)
	//balanceOld, err := GetOtaBalanceFromAX(statedb, otaAX)
	//if err != nil {
	//	return err
	//}
	//
	//if balanceOld != nil && balanceOld.Cmp(common.Big0) != 0 {
	//	return errors.New("ota balance is not 0! old balance:" + balanceOld.String())
	//}

	mptAddr := OTABalance2ContractAddr(balance)
	statedb.SetStateByteArray(mptAddr, common.BytesToHash(otaAX), otaUseAddr)
	return SetOtaBalanceToAX(statedb, otaAX, balance)
}

// AddOTAIfNotExist storage ota info if doesn't exist already.
func AddOTAIfNotExist(statedb StateDB, balance *big.Int, otaUseAddr []byte) (bool, error) {
	if statedb == nil || balance == nil {
		return false, ErrUnknown
	}
	if len(otaUseAddr) != common.UAddressLength {
		return false, ErrInvalidOTAAddr
	}

	otaAX, _ := GetAXFromUseAddr(otaUseAddr)
	otaAddrKey := common.BytesToHash(otaAX)
	exist, _, err := CheckOTAAXExist(statedb, otaAddrKey[:])
	if err != nil {
		return false, err
	}

	if exist {
		return false, ErrOTAExistAlready
	}

	err = setOTA(statedb, balance, otaUseAddr)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetOTAInfoFromAX retrieve ota info, include balance and UseAddr
func GetOTAInfoFromAX(statedb StateDB, otaAX []byte) (otaUseAddr []byte, balance *big.Int, err error) {
	if statedb == nil {
		return nil, nil, ErrUnknown
	}
	if len(otaAX) < common.HashLength {
		return nil, nil, ErrInvalidOTAAX
	}

	otaAddrKey := common.BytesToHash(otaAX)
	balance, err = GetOtaBalanceFromAX(statedb, otaAddrKey[:])
	if err != nil {
		return nil, nil, err
	}

	if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, ErrOTABalanceIsZero
	}

	mptAddr := OTABalance2ContractAddr(balance)

	otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
	if otaValue != nil && len(otaValue) != 0 {
		return otaValue, balance, nil
	}

	return nil, balance, nil
}

type GetOTASetEnv struct {
	otaAX         []byte
	setNum        int
	getNum        int
	loopTimes     int
	rnd           int
	otaUseAddrSet [][]byte
}

func (env *GetOTASetEnv) OTAInSet(ota []byte) bool {
	for _, exist := range env.otaUseAddrSet {
		if bytes.Equal(exist, ota) {
			return true
		}
	}

	return false
}

func (env *GetOTASetEnv) UpdateRnd() {
	env.rnd = rand.Intn(100) + 1
}

func (env *GetOTASetEnv) IsSetFull() bool {
	return env.getNum >= env.setNum
}

func (env *GetOTASetEnv) RandomSelOTA(value []byte) bool {
	env.loopTimes++
	if env.loopTimes%env.rnd == 0 {
		env.otaUseAddrSet = append(env.otaUseAddrSet, value)
		env.getNum++
		env.UpdateRnd()
		return true
	} else {
		return false
	}
}

// doOTAStorageTravelCallBack implement ota mpt travel call back
func doOTAStorageTravelCallBack(env *GetOTASetEnv, value []byte) (bool, error) {
	// find self, return true to continue travel loop
	if IsAXPointToUseAddr(env.otaAX, value) {
		return true, nil
	}

	// ota contained in set already, return true to continue travel loop
	if env.OTAInSet(value) {
		return true, nil
	}

	// random select
	// if set full already, return false to stop travel loop
	if bGet := env.RandomSelOTA(value); bGet {
		return !env.IsSetFull(), nil
	} else {
		return true, nil
	}
}

// GetOTASet retrieve the setNum of same balance OTA address of the input OTA setting by otaAX, and ota balance.
// Rules:
//		1: The result can't contain otaAX self;
//		2: The result can't contain duplicate items;
//		3: No ota exist in the mpt, return error;
//		4: OTA total count in the mpt less or equal to the setNum, return error(returned set must
//		   can't contain otaAX self, so need more exist ota in mpt);
//		5: If find invalid ota Useaddr, return error;
//		6: Travel the ota mpt.Record loop exist ota cumulative times as loopTimes.
// 		   Generate a random number as rnd.
// 		   If loopTimes%rnd == 0, collect current exist ota to result set and update the rnd.
//		   Loop checking exist ota and loop traveling ota mpt, untile collect enough ota or find error.
//
func GetOTASet(statedb StateDB, otaAX []byte, setNum int) (otaUseAddrs [][]byte, balance *big.Int, err error) {
	if statedb == nil {
		return nil, nil, ErrUnknown
	}
	if len(otaAX) != common.HashLength {
		return nil, nil, ErrInvalidOTAAX
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		return nil, nil, err
	} else if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, errors.New("can't find ota address balance!")
	}

	mptAddr := OTABalance2ContractAddr(balance)
	log.Debug("GetOTASet", "mptAddr", common.ToHex(mptAddr[:]))

	env := GetOTASetEnv{otaAX, setNum, 0, 0, 0, nil}
	env.otaUseAddrSet = make([][]byte, 0, setNum)
	env.UpdateRnd()

	mptEleCount := 0 // total number of ota containing in mpt

	for {
		statedb.ForEachStorageByteArray(mptAddr, func(key common.Hash, value []byte) bool {
			mptEleCount++

			if len(value) != common.UAddressLength {
				log.Error("invalid OTA address!", "balance", balance, "value", value)
				err = errors.New(fmt.Sprint("invalid OTA address! balance:", balance, ", ota:", value))
				return false
			}

			bContinue, err := doOTAStorageTravelCallBack(&env, value)
			if err != nil {
				return false
			} else {
				return bContinue
			}
		})

		if env.IsSetFull() {
			return env.otaUseAddrSet, balance, nil
		} else if err != nil {
			return nil, nil, err
		} else if mptEleCount == 0 {
			return nil, balance, errors.New("no ota exist! balance:" + balance.String())
		} else if setNum >= mptEleCount {
			return nil, balance, errors.New("too more required ota number! balance:" + balance.String() +
				", exist count:" + strconv.Itoa(mptEleCount))
		} else {
			continue
		}
	}
}

// CheckOTAImageExist checks ota image key exist already or not
func CheckOTAImageExist(statedb StateDB, otaImage []byte) (bool, []byte, error) {
	if statedb == nil || len(otaImage) == 0 {
		return false, nil, errors.New("invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	otaImageValue := statedb.GetStateByteArray(otaImageStorageAddr, otaImageKey)
	if otaImageValue != nil && len(otaImageValue) != 0 {
		return true, otaImageValue, nil
	}

	return false, nil, nil
}

// AddOTAImage storage ota image key. Overwrite if exist already.
func AddOTAImage(statedb StateDB, otaImage []byte, value []byte) error {
	if statedb == nil || len(otaImage) == 0 || len(value) == 0 {
		return errors.New("invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	statedb.SetStateByteArray(otaImageStorageAddr, otaImageKey, value)
	return nil
}
