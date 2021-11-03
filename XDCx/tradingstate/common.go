package tradingstate

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/crypto"

	"github.com/XinFinOrg/XDPoSChain/common"
)

const (
	OrderCacheLimit = 10000
)

var (
	EmptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	Ask       = "SELL"
	Bid       = "BUY"
	Market    = "MO"
	Limit     = "LO"
	Cancel    = "CANCELLED"
	OrderNew  = "NEW"
)

var EmptyHash = common.Hash{}
var Zero = big.NewInt(0)
var One = big.NewInt(1)

var EmptyOrderList = orderList{
	Volume: nil,
	Root:   EmptyHash,
}
var EmptyExchangeOnject = tradingExchangeObject{
	Nonce:   0,
	AskRoot: EmptyHash,
	BidRoot: EmptyHash,
}
var EmptyOrder = OrderItem{
	Quantity: Zero,
}

var (
	ErrInvalidSignature = errors.New("verify order: invalid signature")
	ErrInvalidPrice     = errors.New("verify order: invalid price")
	ErrInvalidQuantity  = errors.New("verify order: invalid quantity")
	ErrInvalidRelayer   = errors.New("verify order: invalid relayer")
	ErrInvalidOrderType = errors.New("verify order: unsupported order type")
	ErrInvalidOrderSide = errors.New("verify order: invalid order side")
	ErrInvalidStatus    = errors.New("verify order: invalid status")

	// supported order types
	MatchingOrderType = map[string]bool{
		Market: true,
		Limit:  true,
	}
)

// tradingExchangeObject is the Ethereum consensus representation of exchanges.
// These objects are stored in the main orderId trie.
type orderList struct {
	Volume *big.Int
	Root   common.Hash // merkle root of the storage trie
}

// tradingExchangeObject is the Ethereum consensus representation of exchanges.
// These objects are stored in the main orderId trie.
type tradingExchangeObject struct {
	Nonce                  uint64
	LastPrice              *big.Int
	MediumPriceBeforeEpoch *big.Int
	MediumPrice            *big.Int
	TotalQuantity          *big.Int
	LendingCount           *big.Int
	AskRoot                common.Hash // merkle root of the storage trie
	BidRoot                common.Hash // merkle root of the storage trie
	OrderRoot              common.Hash
	LiquidationPriceRoot   common.Hash
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

type TxDataMatch struct {
	Order []byte // serialized data of order has been processed in this tx
}

type TxMatchBatch struct {
	Data      []TxDataMatch
	Timestamp int64
	TxHash    common.Hash
}

type MatchingResult struct {
	Trades  []map[string]string
	Rejects []*OrderItem
}

func EncodeTxMatchesBatch(txMatchBatch TxMatchBatch) ([]byte, error) {
	data, err := json.Marshal(txMatchBatch)
	if err != nil || data == nil {
		return []byte{}, err
	}
	return data, nil
}

func DecodeTxMatchesBatch(data []byte) (TxMatchBatch, error) {
	txMatchResult := TxMatchBatch{}
	if err := json.Unmarshal(data, &txMatchResult); err != nil {
		return TxMatchBatch{}, err
	}
	return txMatchResult, nil
}

// use orderHash instead of orderId
// because both takerOrders don't have orderId
func GetOrderHistoryKey(baseToken, quoteToken common.Address, orderHash common.Hash) common.Hash {
	return crypto.Keccak256Hash(baseToken.Bytes(), quoteToken.Bytes(), orderHash.Bytes())
}

func (tx TxDataMatch) DecodeOrder() (*OrderItem, error) {
	order := &OrderItem{}
	if err := DecodeBytesItem(tx.Order, order); err != nil {
		return order, err
	}
	return order, nil
}

type OrderHistoryItem struct {
	TxHash       common.Hash
	FilledAmount *big.Int
	Status       string
	UpdatedAt    time.Time
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

func GetTradingOrderBookHash(baseToken common.Address, quoteToken common.Address) common.Hash {
	return common.BytesToHash(append(baseToken[:16], quoteToken[4:]...))
}

func GetMatchingResultCacheKey(order *OrderItem) common.Hash {
	return crypto.Keccak256Hash(order.UserAddress.Bytes(), order.Nonce.Bytes())
}
