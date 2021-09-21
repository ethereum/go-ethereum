package core

import (
	"context"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"log"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"
)

type OrderMsg struct {
	AccountNonce    uint64         `json:"nonce"    gencodec:"required"`
	Quantity        *big.Int       `json:"quantity,omitempty"`
	Price           *big.Int       `json:"price,omitempty"`
	ExchangeAddress common.Address `json:"exchangeAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	BaseToken       common.Address `json:"baseToken,omitempty"`
	QuoteToken      common.Address `json:"quoteToken,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	OrderID         uint64         `json:"orderid,omitempty"`
	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

var (
	BTCAddress = common.HexToAddress("0xC2fa1BA90b15E3612E0067A0020192938784D9C5")
	ETHAddress = common.HexToAddress("0xAad540ac542C3688652a3fc7b8e21B3fC1D097e9")
	XRPAddress = common.HexToAddress("0x5dc27D59bB80E0EF853Bb2e27B94113DF08F547F")
	LTCAddress = common.HexToAddress("0x6F98655A8fa7AEEF3147ee002c666d09c7AA4F5c")
	BNBAddress = common.HexToAddress("0xaC389aCA56394a5B14918cF6437600760B6c650C")
	ADAAddress = common.HexToAddress("0x576201Ac3f1E0fe483a9320DaCc4B08EB3E58306")
	ETCAddress = common.HexToAddress("0xf992cf45394dAc5f50A26446de17803a79B940da")
	BCHAddress = common.HexToAddress("0xFDF68dE6dFFd893221fc9f7985FeBC2AB20761A6")
	EOSAddress = common.HexToAddress("0xd9bb01454c85247B2ef35BB5BE57384cC275a8cf")
	USDAddress = common.HexToAddress("0x45c25041b8e6CBD5c963E7943007187C3673C7c9")
	_1E18      = new(big.Int).Mul(big.NewInt(10000000000000000), big.NewInt(100))
	_1E17      = new(big.Int).Mul(big.NewInt(10000000000000000), big.NewInt(10))
	_1E8       = big.NewInt(100000000)
	_1E7       = big.NewInt(10000000)
)

func getNonce(t *testing.T, userAddress common.Address) (uint64, error) {
	rpcClient, err := rpc.DialHTTP("http://127.0.0.1:8501")
	defer rpcClient.Close()
	if err != nil {
		return 0, err
	}
	var result interface{}
	if err != nil {

		return 0, err
	}
	err = rpcClient.Call(&result, "XDCx_getOrderCount", userAddress)
	if err != nil {
		return 0, err
	}
	s := result.(string)
	s = strings.TrimPrefix(s, "0x")
	n, err := strconv.ParseUint(s, 16, 32)
	return uint64(n), nil
}
func testSendOrder(t *testing.T, amount, price *big.Int, side string, status string, orderID uint64) {

	client, err := ethclient.Dial("http://127.0.0.1:8501")
	if err != nil {
		log.Print(err)
	}

	privateKey, err := crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	if err != nil {
		log.Print(err)
	}
	msg := &OrderMsg{
		Quantity:        amount,
		Price:           price,
		ExchangeAddress: common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:     crypto.PubkeyToAddress(privateKey.PublicKey),
		BaseToken:       common.HexToAddress(common.XDCNativeAddress),
		QuoteToken:      BTCAddress,
		Status:          status,
		Side:            side,
		Type:            "LO",
	}
	nonce, _ := getNonce(t, msg.UserAddress)
	tx := types.NewOrderTransaction(nonce, msg.Quantity, msg.Price, msg.ExchangeAddress, msg.UserAddress, msg.BaseToken, msg.QuoteToken, msg.Status, msg.Side, msg.Type, common.Hash{}, orderID)
	signedTx, err := types.OrderSignTx(tx, types.OrderTxSigner{}, privateKey)
	if err != nil {
		log.Print(err)
	}

	err = client.SendOrderTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
	}
}

func testSendOrderXDCUSD(t *testing.T, amount, price *big.Int, side string, status string, orderID uint64) {

	client, err := ethclient.Dial("http://127.0.0.1:8501")
	if err != nil {
		log.Print(err)
	}

	privateKey, err := crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	if err != nil {
		log.Print(err)
	}
	msg := &OrderMsg{
		Quantity:        amount,
		Price:           price,
		ExchangeAddress: common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:     crypto.PubkeyToAddress(privateKey.PublicKey),
		BaseToken:       common.HexToAddress(common.XDCNativeAddress),
		QuoteToken:      USDAddress,
		Status:          status,
		Side:            side,
		Type:            "LO",
	}
	nonce, _ := getNonce(t, msg.UserAddress)
	tx := types.NewOrderTransaction(nonce, msg.Quantity, msg.Price, msg.ExchangeAddress, msg.UserAddress, msg.BaseToken, msg.QuoteToken, msg.Status, msg.Side, msg.Type, common.Hash{}, orderID)
	signedTx, err := types.OrderSignTx(tx, types.OrderTxSigner{}, privateKey)
	if err != nil {
		log.Print(err)
	}

	err = client.SendOrderTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
	}
}

func testSendOrderBTCUSD(t *testing.T, amount, price *big.Int, side string, status string, orderID uint64) {

	client, err := ethclient.Dial("http://127.0.0.1:8501")
	if err != nil {
		log.Print(err)
	}

	privateKey, err := crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	if err != nil {
		log.Print(err)
	}
	msg := &OrderMsg{
		Quantity:        amount,
		Price:           price,
		ExchangeAddress: common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:     crypto.PubkeyToAddress(privateKey.PublicKey),
		BaseToken:       BTCAddress,
		QuoteToken:      USDAddress,
		Status:          status,
		Side:            side,
		Type:            "LO",
	}
	nonce, _ := getNonce(t, msg.UserAddress)
	tx := types.NewOrderTransaction(nonce, msg.Quantity, msg.Price, msg.ExchangeAddress, msg.UserAddress, msg.BaseToken, msg.QuoteToken, msg.Status, msg.Side, msg.Type, common.Hash{}, orderID)
	signedTx, err := types.OrderSignTx(tx, types.OrderTxSigner{}, privateKey)
	if err != nil {
		log.Print(err)
	}

	err = client.SendOrderTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
	}
}

func testSendOrderXDCBTC(t *testing.T, amount, price *big.Int, side string, status string, orderID uint64) {

	client, err := ethclient.Dial("http://127.0.0.1:8501")
	if err != nil {
		log.Print(err)
	}

	privateKey, err := crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	if err != nil {
		log.Print(err)
	}
	msg := &OrderMsg{
		Quantity:        amount,
		Price:           price,
		ExchangeAddress: common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:     crypto.PubkeyToAddress(privateKey.PublicKey),
		BaseToken:       common.HexToAddress(common.XDCNativeAddress),
		QuoteToken:      BTCAddress,
		Status:          status,
		Side:            side,
		Type:            "LO",
	}
	nonce, _ := getNonce(t, msg.UserAddress)
	tx := types.NewOrderTransaction(nonce, msg.Quantity, msg.Price, msg.ExchangeAddress, msg.UserAddress, msg.BaseToken, msg.QuoteToken, msg.Status, msg.Side, msg.Type, common.Hash{}, orderID)
	signedTx, err := types.OrderSignTx(tx, types.OrderTxSigner{}, privateKey)
	if err != nil {
		log.Print(err)
	}

	err = client.SendOrderTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
	}
}

func testSendOrderETHBTC(t *testing.T, amount, price *big.Int, side string, status string, orderID uint64) {

	client, err := ethclient.Dial("http://127.0.0.1:8501")
	if err != nil {
		log.Print(err)
	}

	privateKey, err := crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	if err != nil {
		log.Print(err)
	}
	msg := &OrderMsg{
		Quantity:        amount,
		Price:           price,
		ExchangeAddress: common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:     crypto.PubkeyToAddress(privateKey.PublicKey),
		BaseToken:       ETHAddress,
		QuoteToken:      BTCAddress,
		Status:          status,
		Side:            side,
		Type:            "LO",
	}
	nonce, _ := getNonce(t, msg.UserAddress)
	tx := types.NewOrderTransaction(nonce, msg.Quantity, msg.Price, msg.ExchangeAddress, msg.UserAddress, msg.BaseToken, msg.QuoteToken, msg.Status, msg.Side, msg.Type, common.Hash{}, orderID)
	signedTx, err := types.OrderSignTx(tx, types.OrderTxSigner{}, privateKey)
	if err != nil {
		log.Print(err)
	}

	err = client.SendOrderTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
	}
}

func TestSendBuyOrder(t *testing.T) {
	testSendOrder(t, new(big.Int).SetUint64(1000000000000000000), new(big.Int).SetUint64(100000000000000000), "BUY", "NEW", 0)
}

func TestSendSellOrder(t *testing.T) {
	testSendOrder(t, new(big.Int).SetUint64(1000000000000000000), new(big.Int).SetUint64(100000000000000000), "SELL", "NEW", 0)
}
func TestFilled(t *testing.T) {
	////BTC/XDC
	//BTCUSDPrice := new(big.Int).Mul(big.NewInt(1000000000000000000), big.NewInt(5000))
	//testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(1000000000000000000), big.NewInt(5000)), BTCUSDPrice, "BUY", "NEW", 0)
	//ETH/BTC

	BTCUSDPrice := new(big.Int).Mul(_1E8, big.NewInt(10000)) // 10000
	time.Sleep(2 * time.Second)
	testSendOrderBTCUSD(t, _1E18, BTCUSDPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderBTCUSD(t, _1E18, BTCUSDPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderBTCUSD(t, new(big.Int).Mul(big.NewInt(2), _1E18), BTCUSDPrice, "SELL", "NEW", 0)

	XDCBTCPrice := new(big.Int).Mul(big.NewInt(10000000000000), big.NewInt(6)) // 0.00006
	time.Sleep(2 * time.Second)
	testSendOrderXDCBTC(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCBTCPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderXDCBTC(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCBTCPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderXDCBTC(t, new(big.Int).Mul(big.NewInt(1200000), _1E18), XDCBTCPrice, "SELL", "NEW", 0)

	XDCUSDPrice := new(big.Int).Mul(_1E7, big.NewInt(6)) // 0.6
	time.Sleep(2 * time.Second)
	testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCUSDPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCUSDPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(1200000), _1E18), XDCUSDPrice, "SELL", "NEW", 0)

}

func TestX10Filled(t *testing.T) {
	XDCUSDPrice := new(big.Int).Mul(_1E7, big.NewInt(60)) // 6
	time.Sleep(2 * time.Second)
	testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCUSDPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCUSDPrice, "BUY", "NEW", 0)
	time.Sleep(2 * time.Second)
	testSendOrderXDCUSD(t, new(big.Int).Mul(big.NewInt(1200000), _1E18), XDCUSDPrice, "SELL", "NEW", 0)

}
func TestPartialFilled(t *testing.T) {

}
func TestNoMatch(t *testing.T) {

}

func TestCancelOrder(t *testing.T) {
	XDCBTCPrice := new(big.Int).Mul(big.NewInt(10000000000000), big.NewInt(6)) // 0.00006
	testSendOrder(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCBTCPrice, "BUY", "NEW", 0)
	time.Sleep(5 * time.Second)
	testSendOrder(t, new(big.Int).Mul(big.NewInt(600000), _1E18), XDCBTCPrice, "BUY", "CANCELLED", 3)
	time.Sleep(5 * time.Second)
	//testSendOrder(t, new(big.Int).SetUint64(48), new(big.Int).SetUint64(15), "SELL", "NEW", 0)
}
