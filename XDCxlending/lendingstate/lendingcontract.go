package lendingstate

import (
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
	"math/big"
)

var (
	LendingRelayerListSlot    = uint64(0)
	CollateralMapSlot         = uint64(1)
	DefaultCollateralSlot     = uint64(2)
	SupportedBaseSlot         = uint64(3)
	SupportedTermSlot         = uint64(4)
	ILOCollateralSlot         = uint64(5)
	LendingRelayerStructSlots = map[string]*big.Int{
		"fee":         big.NewInt(0),
		"bases":       big.NewInt(1),
		"terms":       big.NewInt(2),
		"collaterals": big.NewInt(3),
	}
	CollateralStructSlots = map[string]*big.Int{
		"depositRate":     big.NewInt(0),
		"liquidationRate": big.NewInt(1),
		"recallRate":      big.NewInt(2),
		"price":           big.NewInt(3),
	}
	PriceStructSlots = map[string]*big.Int{
		"price":       big.NewInt(0),
		"blockNumber": big.NewInt(1),
	}
)

// @function IsValidRelayer : return whether the given address is the coinbase of a valid relayer or not
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: true if it's a valid coinbase address of lending protocol, otherwise return false
func IsValidRelayer(statedb *state.StateDB, coinbase common.Address) bool {
	locRelayerState := GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)

	// a valid relayer must have baseToken
	locBaseToken := state.GetLocOfStructElement(locRelayerState, LendingRelayerStructSlots["bases"])
	if v := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), common.BytesToHash(locBaseToken.Bytes())); v != (common.Hash{}) {
		if tradingstate.IsResignedRelayer(coinbase, statedb) {
			return false
		}
		slot := tradingstate.RelayerMappingSlot["RELAYER_LIST"]
		locRelayerStateTrading := GetLocMappingAtKey(coinbase.Hash(), slot)

		locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locRelayerStateTrading, tradingstate.RelayerStructMappingSlot["_deposit"])
		locHashDeposit := common.BigToHash(locBigDeposit)
		balance := statedb.GetState(common.HexToAddress(common.RelayerRegistrationSMC), locHashDeposit).Big()
		expectedFund := new(big.Int).Mul(common.BasePrice, common.RelayerLockedFund)
		if balance.Cmp(expectedFund) <= 0 {
			log.Debug("Relayer is not in relayer list", "relayer", coinbase.String(), "balance", balance, "expected", expectedFund)
			return false
		}
		return true
	}
	return false
}

// @function GetFee
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: feeRate of lending
func GetFee(statedb *state.StateDB, coinbase common.Address) *big.Int {
	locRelayerState := state.GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locHash := common.BytesToHash(new(big.Int).Add(locRelayerState, LendingRelayerStructSlots["fee"]).Bytes())
	return statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locHash).Big()
}

// @function GetBaseList
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: list of base tokens
func GetBaseList(statedb *state.StateDB, coinbase common.Address) []common.Address {
	baseList := []common.Address{}
	locRelayerState := state.GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locBaseHash := state.GetLocOfStructElement(locRelayerState, LendingRelayerStructSlots["bases"])
	length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locBaseHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := state.GetLocDynamicArrAtElement(locBaseHash, i, 1)
		addr := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Bytes())
		if addr != (common.Address{}) {
			baseList = append(baseList, addr)
		}
	}
	return baseList
}

// @function GetTerms
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: list of supported terms of the given relayer
func GetTerms(statedb *state.StateDB, coinbase common.Address) []uint64 {
	terms := []uint64{}
	locRelayerState := state.GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locTermHash := state.GetLocOfStructElement(locRelayerState, LendingRelayerStructSlots["terms"])
	length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locTermHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := state.GetLocDynamicArrAtElement(locTermHash, i, 1)
		t := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Big().Uint64()
		if t != uint64(0) {
			terms = append(terms, t)
		}
	}
	return terms
}

// @function IsValidPair
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @param baseToken: address of baseToken
// @param terms: term
// @return: TRUE if the given baseToken, term organize a valid pair
func IsValidPair(statedb *state.StateDB, coinbase common.Address, baseToken common.Address, term uint64) (valid bool, pairIndex uint64) {
	baseTokenList := GetBaseList(statedb, coinbase)
	terms := GetTerms(statedb, coinbase)
	baseIndexes := []uint64{}
	for i := uint64(0); i < uint64(len(baseTokenList)); i++ {
		if baseTokenList[i] == baseToken {
			baseIndexes = append(baseIndexes, i)
		}
	}
	for _, index := range baseIndexes {
		if terms[index] == term {
			pairIndex = index
			return true, pairIndex
		}
	}
	return false, pairIndex
}

// @function GetCollaterals
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @param baseToken: address of baseToken
// @param terms: term
// @return:
//		- collaterals []common.Address  : list of addresses of collateral
//		- isSpecialCollateral			: TRUE if collateral is a token which is NOT available for trading in XDCX, otherwise FALSE
func GetCollaterals(statedb *state.StateDB, coinbase common.Address, baseToken common.Address, term uint64) (collaterals []common.Address, isSpecialCollateral bool) {
	validPair, _ := IsValidPair(statedb, coinbase, baseToken, term)
	if !validPair {
		return []common.Address{}, false
	}

	//TODO: ILO Collateral is not supported in release 2.2.0
	//locRelayerState := state.GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	//locCollateralHash := state.GetLocOfStructElement(locRelayerState, LendingRelayerStructSlots["collaterals"])
	//length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locCollateralHash).Big().Uint64()
	//
	//loc := state.GetLocDynamicArrAtElement(locCollateralHash, pairIndex, 1)
	//collateralAddr := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Bytes())
	//if collateralAddr != (common.Address{}) && collateralAddr != (common.HexToAddress("0x0")) {
	//	return []common.Address{collateralAddr}, true
	//}

	// if collaterals is not defined for the relayer, return default collaterals
	locDefaultCollateralHash := state.GetLocSimpleVariable(DefaultCollateralSlot)
	length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locDefaultCollateralHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := state.GetLocDynamicArrAtElement(locDefaultCollateralHash, i, 1)
		addr := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Bytes())
		if addr != (common.Address{}) {
			collaterals = append(collaterals, addr)
		}
	}
	return collaterals, false
}

// @function GetCollateralDetail
// @param statedb : current state
// @param token: address of collateral token
// @return: depositRate, liquidationRate, price of collateral
func GetCollateralDetail(statedb *state.StateDB, token common.Address) (depositRate, liquidationRate, recallRate *big.Int) {
	collateralState := GetLocMappingAtKey(token.Hash(), CollateralMapSlot)
	locDepositRate := state.GetLocOfStructElement(collateralState, CollateralStructSlots["depositRate"])
	locLiquidationRate := state.GetLocOfStructElement(collateralState, CollateralStructSlots["liquidationRate"])
	locRecallRate := state.GetLocOfStructElement(collateralState, CollateralStructSlots["recallRate"])
	depositRate = statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locDepositRate).Big()
	liquidationRate = statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locLiquidationRate).Big()
	recallRate = statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locRecallRate).Big()
	return depositRate, liquidationRate, recallRate
}

func GetCollateralPrice(statedb *state.StateDB, collateralToken common.Address, lendingToken common.Address) (price, blockNumber *big.Int) {
	collateralState := GetLocMappingAtKey(collateralToken.Hash(), CollateralMapSlot)
	locMapPrices := collateralState.Add(collateralState, CollateralStructSlots["price"])
	locLendingTokenPriceByte := crypto.Keccak256(lendingToken.Hash().Bytes(), common.BigToHash(locMapPrices).Bytes())

	locCollateralPrice := common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(locLendingTokenPriceByte), PriceStructSlots["price"]))
	locBlockNumber := common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(locLendingTokenPriceByte), PriceStructSlots["blockNumber"]))

	price = statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locCollateralPrice).Big()
	blockNumber = statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locBlockNumber).Big()
	return price, blockNumber
}

// @function GetSupportedTerms
// @param statedb : current state
// @return: list of terms which XDCxlending supports
func GetSupportedTerms(statedb *state.StateDB) []uint64 {
	terms := []uint64{}
	locSupportedTerm := state.GetLocSimpleVariable(SupportedTermSlot)
	length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locSupportedTerm).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := state.GetLocDynamicArrAtElement(locSupportedTerm, i, 1)
		t := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Big().Uint64()
		if t != 0 {
			terms = append(terms, t)
		}
	}
	return terms
}

// @function GetSupportedBaseToken
// @param statedb : current state
// @return: list of tokens which are available for lending
func GetSupportedBaseToken(statedb *state.StateDB) []common.Address {
	baseTokens := []common.Address{}
	locSupportedBaseToken := state.GetLocSimpleVariable(SupportedBaseSlot)
	length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locSupportedBaseToken).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := state.GetLocDynamicArrAtElement(locSupportedBaseToken, i, 1)
		addr := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Bytes())
		if addr != (common.Address{}) {
			baseTokens = append(baseTokens, addr)
		}
	}
	return baseTokens
}

// @function GetAllCollateral
// @param statedb : current state
// @return: list of address of collateral token
func GetAllCollateral(statedb *state.StateDB) []common.Address {
	collaterals := []common.Address{}

	//TODO: ILO Collateral is not supported in release 2.2.0
	//locILOCollateral := state.GetLocSimpleVariable(ILOCollateralSlot)
	//length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locILOCollateral).Big().Uint64()
	//for i := uint64(0); i < length; i++ {
	//	loc := state.GetLocDynamicArrAtElement(locILOCollateral, i, 1)
	//	addr := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Bytes())
	//	if addr != (common.Address{}) {
	//		collaterals = append(collaterals, addr)
	//	}
	//}

	locDefaultCollateralHash := state.GetLocSimpleVariable(DefaultCollateralSlot)
	length := statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), locDefaultCollateralHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := state.GetLocDynamicArrAtElement(locDefaultCollateralHash, i, 1)
		addr := common.BytesToAddress(statedb.GetState(common.HexToAddress(common.LendingRegistrationSMC), loc).Bytes())
		if addr != (common.Address{}) {
			collaterals = append(collaterals, addr)
		}
	}
	return collaterals
}

// @function GetAllLendingBooks
// @param statedb : current state
// @return: a map to specify whether lendingBook (combination of baseToken and term) is valid or not
func GetAllLendingBooks(statedb *state.StateDB) (mapLendingBook map[common.Hash]bool, err error) {
	mapLendingBook = make(map[common.Hash]bool)
	baseTokens := GetSupportedBaseToken(statedb)
	terms := GetSupportedTerms(statedb)
	if len(baseTokens) == 0 {
		return nil, fmt.Errorf("GetAllLendingBooks: empty baseToken list")
	}
	if len(terms) == 0 {
		return nil, fmt.Errorf("GetAllLendingPairs: empty term list")
	}
	for _, baseToken := range baseTokens {
		for _, term := range terms {
			if (baseToken != common.Address{}) && (term > 0) {
				mapLendingBook[GetLendingOrderBookHash(baseToken, term)] = true
			}
		}
	}
	return mapLendingBook, nil
}

// @function GetAllLendingPairs
// @param statedb : current state
// @return: list of lendingPair (combination of baseToken and collateralToken)
func GetAllLendingPairs(statedb *state.StateDB) (allPairs []LendingPair, err error) {
	baseTokens := GetSupportedBaseToken(statedb)
	collaterals := GetAllCollateral(statedb)
	if len(baseTokens) == 0 {
		return allPairs, fmt.Errorf("GetAllLendingPairs: empty baseToken list")
	}
	if len(collaterals) == 0 {
		return allPairs, fmt.Errorf("GetAllLendingPairs: empty collateral list")
	}
	for _, baseToken := range baseTokens {
		for _, collateral := range collaterals {
			if (baseToken != common.Address{}) && (collateral != common.Address{}) {
				allPairs = append(allPairs, LendingPair{
					LendingToken:    baseToken,
					CollateralToken: collateral,
				})
			}
		}
	}
	return allPairs, nil
}
