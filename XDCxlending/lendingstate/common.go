package lendingstate

import (
	"encoding/json"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
)

var (
	EmptyAddress = "0x0000000000000000000000000000000000000000"
	EmptyRoot    = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

var EmptyHash = common.Hash{}
var Zero = big.NewInt(0)
var One = big.NewInt(1)
var EmptyLendingOrder = LendingItem{
	Quantity: Zero,
}

var EmptyLendingTrade = LendingTrade{
	Amount: big.NewInt(0),
}

type itemList struct {
	Volume *big.Int
	Root   common.Hash
}

type lendingObject struct {
	Nonce               uint64
	TradeNonce          uint64
	InvestingRoot       common.Hash
	BorrowingRoot       common.Hash
	LiquidationTimeRoot common.Hash
	LendingItemRoot     common.Hash
	LendingTradeRoot    common.Hash
}

// liquidation reasons
const (
	LiquidatedByTime  = uint64(0)
	LiquidatedByPrice = uint64(1)
)

type LiquidationData struct {
	RecallAmount      *big.Int
	LiquidationAmount *big.Int
	CollateralPrice   *big.Int
	Reason            uint64
}

var (
	TokenMappingSlot = map[string]uint64{
		"balances": 0,
	}
	RelayerMappingSlot = map[string]uint64{
		"CONTRACT_OWNER":       0,
		"MaximumRelayers":      1,
		"MaximumTokenList":     2,
		"RELAYER_LIST":         3,
		"RELAYER_COINBASES":    4,
		"RESIGN_REQUESTS":      5,
		"RELAYER_ON_SALE_LIST": 6,
		"RelayerCount":         7,
		"MinimumDeposit":       8,
	}
	RelayerStructMappingSlot = map[string]*big.Int{
		"_deposit":    big.NewInt(0),
		"_fee":        big.NewInt(1),
		"_fromTokens": big.NewInt(2),
		"_toTokens":   big.NewInt(3),
		"_index":      big.NewInt(4),
		"_owner":      big.NewInt(5),
	}
)

type TxLendingBatch struct {
	Data      []*LendingItem
	Timestamp int64
	TxHash    common.Hash
}

type MatchingResult struct {
	Trades  []*LendingTrade
	Rejects []*LendingItem
}

type FinalizedResult struct {
	Liquidated []common.Hash
	AutoRepay  []common.Hash
	AutoTopUp  []common.Hash
	AutoRecall []common.Hash
	TxHash     common.Hash
	Timestamp  int64
}

// use orderHash instead of tradeId
// because both takerOrders don't have tradeId
func GetLendingItemHistoryKey(lendingToken, collateralToken common.Address, lendingItemHash common.Hash) common.Hash {
	return crypto.Keccak256Hash(lendingToken.Bytes(), collateralToken.Bytes(), lendingItemHash.Bytes())
}

type LendingItemHistoryItem struct {
	TxHash       common.Hash
	FilledAmount *big.Int
	Status       string
	UpdatedAt    time.Time
}

type LendingTradeHistoryItem struct {
	TxHash                 common.Hash
	CollateralLockedAmount *big.Int
	LiquidationPrice       *big.Int
	Status                 string
	UpdatedAt              time.Time
}

type LendingPair struct {
	LendingToken    common.Address
	CollateralToken common.Address
}

// ToJSON : log json string
func ToJSON(object interface{}, args ...string) string {
	var str []byte
	if len(args) == 2 {
		str, _ = json.MarshalIndent(object, args[0], args[1])
	} else {
		str, _ = json.Marshal(object)
	}
	return string(str)
}

func Mul(x, y *big.Int) *big.Int {
	return big.NewInt(0).Mul(x, y)
}

func Div(x, y *big.Int) *big.Int {
	return big.NewInt(0).Div(x, y)
}

func Add(x, y *big.Int) *big.Int {
	return big.NewInt(0).Add(x, y)
}

func Sub(x, y *big.Int) *big.Int {
	return big.NewInt(0).Sub(x, y)
}

func Neg(x *big.Int) *big.Int {
	return big.NewInt(0).Neg(x)
}

func ToBigInt(s string) *big.Int {
	res := big.NewInt(0)
	res.SetString(s, 10)
	return res
}

func CloneBigInt(bigInt *big.Int) *big.Int {
	res := new(big.Int).SetBytes(bigInt.Bytes())
	return res
}

func Exp(x, y *big.Int) *big.Int {
	return big.NewInt(0).Exp(x, y, nil)
}

func Max(a, b *big.Int) *big.Int {
	if a.Cmp(b) == 1 {
		return a
	} else {
		return b
	}
}

func GetLendingOrderBookHash(lendingToken common.Address, term uint64) common.Hash {
	return crypto.Keccak256Hash(append(common.Uint64ToHash(term).Bytes(), lendingToken.Bytes()...))
}

func EncodeTxLendingBatch(batch TxLendingBatch) ([]byte, error) {
	data, err := json.Marshal(batch)
	if err != nil || data == nil {
		return []byte{}, err
	}
	return data, nil
}

func DecodeTxLendingBatch(data []byte) (TxLendingBatch, error) {
	txMatchResult := TxLendingBatch{}
	if err := json.Unmarshal(data, &txMatchResult); err != nil {
		return TxLendingBatch{}, err
	}
	return txMatchResult, nil
}

func EtherToWei(amount *big.Int) *big.Int {
	return new(big.Int).Mul(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

func WeiToEther(amount *big.Int) *big.Int {
	return new(big.Int).Div(amount, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

func EncodeFinalizedResult(liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades []*LendingTrade) ([]byte, error) {
	liquidatedHashes := []common.Hash{}
	autoRepayHashes := []common.Hash{}
	autoTopUpHashes := []common.Hash{}
	autoRecallHashes := []common.Hash{}

	for _, trade := range liquidatedTrades {
		liquidatedHashes = append(liquidatedHashes, trade.Hash)
	}
	for _, trade := range autoRepayTrades {
		autoRepayHashes = append(autoRepayHashes, trade.Hash)
	}
	for _, trade := range autoTopUpTrades {
		autoTopUpHashes = append(autoTopUpHashes, trade.Hash)
	}
	for _, trade := range autoRecallTrades {
		autoRecallHashes = append(autoRecallHashes, trade.Hash)
	}
	result := FinalizedResult{
		Liquidated: liquidatedHashes,
		AutoRepay:  autoRepayHashes,
		AutoTopUp:  autoTopUpHashes,
		AutoRecall: autoRecallHashes,
		Timestamp:  time.Now().UnixNano(),
	}
	data, err := json.Marshal(result)
	if err != nil || data == nil {
		return []byte{}, err
	}
	return data, nil
}

func DecodeFinalizedResult(data []byte) (result FinalizedResult, err error) {
	result = FinalizedResult{}
	if err := json.Unmarshal(data, &result); err != nil {
		return FinalizedResult{}, err
	}
	return result, nil
}

func GetLendingCacheKey(item *LendingItem) common.Hash {
	return crypto.Keccak256Hash(item.UserAddress.Bytes(), item.Nonce.Bytes())
}
