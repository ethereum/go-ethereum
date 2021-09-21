package tradingstate

import (
	"fmt"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/pkg/errors"
)

func GetLocMappingAtKey(key common.Hash, slot uint64) *big.Int {
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	retByte := crypto.Keccak256(key.Bytes(), slotHash.Bytes())
	ret := new(big.Int)
	ret.SetBytes(retByte)
	return ret
}

func GetExRelayerFee(relayer common.Address, statedb *state.StateDB) *big.Int {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_fee"])
	locHash := common.BigToHash(locBig)
	return statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHash).Big()
}

func GetRelayerOwner(relayer common.Address, statedb *state.StateDB) common.Address {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	log.Debug("GetRelayerOwner", "relayer", relayer.Hex(), "slot", slot, "locBig", locBig)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_owner"])
	locHash := common.BigToHash(locBig)
	return common.BytesToAddress(statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHash).Bytes())
}

// return true if relayer request to resign and have not withdraw locked fund
func IsResignedRelayer(relayer common.Address, statedb *state.StateDB) bool {
	slot := RelayerMappingSlot["RESIGN_REQUESTS"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locHash := common.BigToHash(locBig)
	if statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHash) != (common.Hash{}) {
		return true
	}
	return false
}

func GetBaseTokenLength(relayer common.Address, statedb *state.StateDB) uint64 {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_fromTokens"])
	locHash := common.BigToHash(locBig)
	return statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHash).Big().Uint64()
}

func GetBaseTokenAtIndex(relayer common.Address, statedb *state.StateDB, index uint64) common.Address {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_fromTokens"])
	locHash := common.BigToHash(locBig)
	loc := state.GetLocDynamicArrAtElement(locHash, index, 1)
	return common.BytesToAddress(statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), loc).Bytes())
}

func GetQuoteTokenLength(relayer common.Address, statedb *state.StateDB) uint64 {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_toTokens"])
	locHash := common.BigToHash(locBig)
	return statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHash).Big().Uint64()
}

func GetQuoteTokenAtIndex(relayer common.Address, statedb *state.StateDB, index uint64) common.Address {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_toTokens"])
	locHash := common.BigToHash(locBig)
	loc := state.GetLocDynamicArrAtElement(locHash, index, 1)
	return common.BytesToAddress(statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), loc).Bytes())
}

func GetRelayerCount(statedb *state.StateDB) uint64 {
	slot := RelayerMappingSlot["RelayerCount"]
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	valueHash := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), slotHash)
	return new(big.Int).SetBytes(valueHash.Bytes()).Uint64()
}

func GetAllCoinbases(statedb *state.StateDB) []common.Address {
	relayerCount := GetRelayerCount(statedb)
	slot := RelayerMappingSlot["RELAYER_COINBASES"]
	coinbases := []common.Address{}
	for i := uint64(0); i < relayerCount; i++ {
		valueHash := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), common.BytesToHash(state.GetLocMappingAtKey(common.BigToHash(big.NewInt(int64(i))), slot).Bytes()))
		coinbases = append(coinbases, common.BytesToAddress(valueHash.Bytes()))
	}
	return coinbases
}
func GetAllTradingPairs(statedb *state.StateDB) (map[common.Hash]bool, error) {
	coinbases := GetAllCoinbases(statedb)
	slot := RelayerMappingSlot["RELAYER_LIST"]
	allPairs := map[common.Hash]bool{}
	for _, coinbase := range coinbases {
		locBig := GetLocMappingAtKey(coinbase.Hash(), slot)
		fromTokenSlot := new(big.Int).Add(locBig, RelayerStructMappingSlot["_fromTokens"])
		fromTokenLength := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), common.BigToHash(fromTokenSlot)).Big().Uint64()
		toTokenSlot := new(big.Int).Add(locBig, RelayerStructMappingSlot["_toTokens"])
		toTokenLength := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), common.BigToHash(toTokenSlot)).Big().Uint64()
		if toTokenLength != fromTokenLength {
			return map[common.Hash]bool{}, fmt.Errorf("Invalid length from token & to toke : from :%d , to :%d ", fromTokenLength, toTokenLength)
		}
		fromTokens := []common.Address{}
		fromTokenSlotHash := common.BytesToHash(fromTokenSlot.Bytes())
		for i := uint64(0); i < fromTokenLength; i++ {
			fromToken := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), state.GetLocDynamicArrAtElement(fromTokenSlotHash, i, uint64(1))).Bytes())
			fromTokens = append(fromTokens, fromToken)
		}
		toTokenSlotHash := common.BytesToHash(toTokenSlot.Bytes())
		for i := uint64(0); i < toTokenLength; i++ {
			toToken := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), state.GetLocDynamicArrAtElement(toTokenSlotHash, i, uint64(1))).Bytes())

			log.Debug("GetAllTradingPairs all pair info", "from", fromTokens[i].Hex(), "toToken", toToken.Hex())
			allPairs[GetTradingOrderBookHash(fromTokens[i], toToken)] = true
		}
	}
	log.Debug("GetAllTradingPairs", "coinbase", len(coinbases), "allPairs", len(allPairs))
	return allPairs, nil
}

func SubRelayerFee(relayer common.Address, fee *big.Int, statedb *state.StateDB) error {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)

	locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
	locHashDeposit := common.BigToHash(locBigDeposit)
	balance := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHashDeposit).Big()
	log.Debug("ApplyXDCXMatchedTransaction settle balance: SubRelayerFee BEFORE", "relayer", relayer.String(), "balance", balance)
	if balance.Cmp(fee) < 0 {
		return errors.Errorf("relayer %s isn't enough XDC fee", relayer.String())
	} else {
		balance = new(big.Int).Sub(balance, fee)
		statedb.SetState(common.HexToAddress(common.RelayerRegistrationSMC), locHashDeposit, common.BigToHash(balance))
		statedb.SubBalance(common.HexToAddress(common.RelayerRegistrationSMC), fee)
		log.Debug("ApplyXDCXMatchedTransaction settle balance: SubRelayerFee AFTER", "relayer", relayer.String(), "balance", balance)
		return nil
	}
}

func CheckRelayerFee(relayer common.Address, fee *big.Int, statedb *state.StateDB) error {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)

	locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
	locHashDeposit := common.BigToHash(locBigDeposit)
	balance := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHashDeposit).Big()
	if new(big.Int).Sub(balance, fee).Cmp(new(big.Int).Mul(common.BasePrice, common.RelayerLockedFund)) < 0 {
		return errors.Errorf("relayer %s isn't enough XDC fee : balance %d , fee : %d ", relayer.Hex(), balance.Uint64(), fee.Uint64())
	}
	return nil
}
func AddTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB) error {
	// XDC native
	if token.String() == common.XDCNativeAddress {
		balance := statedb.GetBalance(addr)
		log.Debug("ApplyXDCXMatchedTransaction settle balance: ADD TOKEN XDC NATIVE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		statedb.AddBalance(addr, value)
		balance = statedb.GetBalance(addr)
		log.Debug("ApplyXDCXMatchedTransaction settle balance: ADD XDC NATIVE BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)

		return nil
	}

	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		balance := statedb.GetState(token, locHash).Big()
		log.Debug("ApplyXDCXMatchedTransaction settle balance: ADD TOKEN BALANCE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		balance = new(big.Int).Add(balance, value)
		statedb.SetState(token, locHash, common.BigToHash(balance))
		log.Debug("ApplyXDCXMatchedTransaction settle balance: ADD TOKEN BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		return nil
	} else {
		return errors.Errorf("token %s isn't exist", token.String())
	}
}

func SubTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB) error {
	// XDC native
	if token.String() == common.XDCNativeAddress {

		balance := statedb.GetBalance(addr)
		log.Debug("ApplyXDCXMatchedTransaction settle balance: SUB XDC NATIVE BALANCE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		if balance.Cmp(value) < 0 {
			return errors.Errorf("value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		statedb.SubBalance(addr, value)
		balance = statedb.GetBalance(addr)
		log.Debug("ApplyXDCXMatchedTransaction settle balance: SUB XDC NATIVE BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		return nil
	}

	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		balance := statedb.GetState(token, locHash).Big()
		log.Debug("ApplyXDCXMatchedTransaction settle balance: SUB TOKEN BALANCE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		if balance.Cmp(value) < 0 {
			return errors.Errorf("value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		balance = new(big.Int).Sub(balance, value)
		statedb.SetState(token, locHash, common.BigToHash(balance))
		log.Debug("ApplyXDCXMatchedTransaction settle balance: SUB TOKEN BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		return nil
	} else {
		return errors.Errorf("token %s isn't exist", token.String())
	}
}

func CheckSubTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB, mapBalances map[common.Address]map[common.Address]*big.Int) (*big.Int, error) {
	// XDC native
	if token.String() == common.XDCNativeAddress {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			balance = statedb.GetBalance(addr)
		}
		if balance.Cmp(value) < 0 {
			return nil, errors.Errorf("value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		newBalance := new(big.Int).Sub(balance, value)
		log.Debug("CheckSubTokenBalance settle balance: SUB XDC NATIVE BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		return newBalance, nil
	}
	// TRC tokens
	if statedb.Exist(token) {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			slot := TokenMappingSlot["balances"]
			locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
			balance = statedb.GetState(token, locHash).Big()
		}
		if balance.Cmp(value) < 0 {
			return nil, errors.Errorf("value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		newBalance := new(big.Int).Sub(balance, value)
		log.Debug("CheckSubTokenBalance settle balance: SUB TOKEN BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		return newBalance, nil
	} else {
		return nil, errors.Errorf("token %s isn't exist", token.String())
	}
}

func CheckAddTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB, mapBalances map[common.Address]map[common.Address]*big.Int) (*big.Int, error) {
	// XDC native
	if token.String() == common.XDCNativeAddress {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			balance = statedb.GetBalance(addr)
		}
		newBalance := new(big.Int).Add(balance, value)
		log.Debug("CheckAddTokenBalance settle balance: ADD XDC NATIVE BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		return newBalance, nil
	}
	// TRC tokens
	if statedb.Exist(token) {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			slot := TokenMappingSlot["balances"]
			locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
			balance = statedb.GetState(token, locHash).Big()
		}
		newBalance := new(big.Int).Add(balance, value)
		log.Debug("CheckAddTokenBalance settle balance: ADD TOKEN BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		if common.BigToHash(newBalance).Big().Cmp(newBalance) != 0 {
			return nil, fmt.Errorf("Overflow when try add token balance , max is 2^256 , balance : %v , value:%v ", balance, value)
		} else {
			return newBalance, nil
		}
	} else {
		return nil, errors.Errorf("token %s isn't exist", token.String())
	}
}

func CheckSubRelayerFee(relayer common.Address, fee *big.Int, statedb *state.StateDB, mapBalances map[common.Address]*big.Int) (*big.Int, error) {
	balance := mapBalances[relayer]
	if balance == nil {
		slot := RelayerMappingSlot["RELAYER_LIST"]
		locBig := GetLocMappingAtKey(relayer.Hash(), slot)
		locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
		locHashDeposit := common.BigToHash(locBigDeposit)
		balance = statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHashDeposit).Big()
	}
	log.Debug("CheckSubRelayerFee settle balance: SubRelayerFee ", "relayer", relayer.String(), "balance", balance, "fee", fee)
	if balance.Cmp(fee) < 0 {
		return nil, errors.Errorf("relayer %s isn't enough XDC fee", relayer.String())
	} else {
		return new(big.Int).Sub(balance, fee), nil
	}
}

func GetTokenBalance(addr common.Address, token common.Address, statedb *state.StateDB) *big.Int {
	// XDC native
	if token.String() == common.XDCNativeAddress {
		return statedb.GetBalance(addr)
	}
	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		return statedb.GetState(token, locHash).Big()
	} else {
		return common.Big0
	}
}

func SetTokenBalance(addr common.Address, balance *big.Int, token common.Address, statedb *state.StateDB) error {
	// XDC native
	if token.String() == common.XDCNativeAddress {
		statedb.SetBalance(addr, balance)
		return nil
	}

	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		statedb.SetState(token, locHash, common.BigToHash(balance))
		return nil
	} else {
		return errors.Errorf("token %s isn't exist", token.String())
	}
}

func SetSubRelayerFee(relayer common.Address, balance *big.Int, fee *big.Int, statedb *state.StateDB) {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
	locHashDeposit := common.BigToHash(locBigDeposit)
	statedb.SetState(common.HexToAddress(common.RelayerRegistrationSMC), locHashDeposit, common.BigToHash(balance))
	statedb.SubBalance(common.HexToAddress(common.RelayerRegistrationSMC), fee)
}
