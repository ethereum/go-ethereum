package core

import (
	"context"
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"log"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"
)

type LendingMsg struct {
	AccountNonce    uint64         `json:"nonce"    gencodec:"required"`
	Quantity        *big.Int       `json:"quantity,omitempty"`
	RelayerAddress  common.Address `json:"relayerAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	CollateralToken common.Address `json:"collateralToken,omitempty"`
	AutoTopUp       bool           `json:"autoTopUp,omitempty"`
	LendingToken    common.Address `json:"lendingToken,omitempty"`
	Term            uint64         `json:"term,omitempty"`
	Interest        uint64         `json:"interest,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	LendingId       uint64         `json:"lendingId,omitempty"`
	LendingTradeId  uint64         `json:"tradeId,omitempty"`
	ExtraData       string         `json:"extraData,omitempty"`
	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

func getLendingNonce(userAddress common.Address) (uint64, error) {
	rpcClient, err := rpc.DialHTTP("http://127.0.0.1:8501")
	defer rpcClient.Close()
	if err != nil {
		return 0, err
	}
	var result interface{}
	err = rpcClient.Call(&result, "XDCx_getLendingOrderCount", userAddress)
	if err != nil {
		return 0, err
	}
	s := result.(string)
	s = strings.TrimPrefix(s, "0x")
	n, err := strconv.ParseUint(s, 16, 32)
	return uint64(n), nil
}

func (l *LendingMsg) computeHash() common.Hash {
	borrowing := l.Side == lendingstate.Borrowing
	sha := sha3.NewKeccak256()
	if l.Type == lendingstate.Repay {
		sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.RelayerAddress.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingTradeId))).Bytes())
	} else if l.Type == lendingstate.TopUp {
		sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.RelayerAddress.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingTradeId))).Bytes())
		sha.Write(common.BigToHash(l.Quantity).Bytes())
	} else {
		if l.Status == lendingstate.LendingStatusCancelled {
			sha := sha3.NewKeccak256()
			sha.Write(l.Hash.Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
			sha.Write(l.UserAddress.Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.LendingId))).Bytes())
			sha.Write([]byte(l.Status))
			sha.Write(l.RelayerAddress.Bytes())
		} else if l.Status == lendingstate.LendingStatusNew {
			sha.Write(l.RelayerAddress.Bytes())
			sha.Write(l.UserAddress.Bytes())
			if borrowing {
				sha.Write(l.CollateralToken.Bytes())
			}
			sha.Write(l.LendingToken.Bytes())
			sha.Write(common.BigToHash(l.Quantity).Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
			if l.Type == lendingstate.Limit {
				sha.Write(common.BigToHash(big.NewInt(int64(l.Interest))).Bytes())
			}
			sha.Write([]byte(l.Side))
			sha.Write([]byte(l.Status))
			sha.Write([]byte(l.Type))
			sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.LendingTradeId))).Bytes())
			if borrowing {
				autoTopUp := int64(0)
				if l.AutoTopUp {
					autoTopUp = int64(1)
				}
				sha.Write(common.BigToHash(big.NewInt(autoTopUp)).Bytes())
			}
		}
	}

	return common.BytesToHash(sha.Sum(nil))

}
func testSendLending(key string, nonce uint64, lendToken, collateralToken common.Address, amount *big.Int, interest uint64, side string, status string, autoTopUp bool, lendingId, tradeId uint64, cancelledHash common.Hash, extraData string) {

	client, err := ethclient.Dial("http://127.0.0.1:8501")
	if err != nil {
		log.Print(err)
	}
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Print(err)
	}
	msg := &LendingMsg{
		AccountNonce:   nonce,
		Quantity:       amount,
		RelayerAddress: common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:    crypto.PubkeyToAddress(privateKey.PublicKey),
		LendingToken:   lendToken,
		Status:         status,
		Side:           side,
		Type:           "LO",
		Term:           86400,
		AutoTopUp:      autoTopUp,
		Interest:       interest,
		LendingId:      lendingId,
		LendingTradeId: tradeId,
		ExtraData:      extraData,
	}
	if msg.Side == lendingstate.Borrowing {
		msg.CollateralToken = collateralToken
	}
	if cancelledHash != (common.Hash{}) {
		msg.Hash = cancelledHash
	} else {
		msg.Hash = msg.computeHash()
	}

	tx := types.NewLendingTransaction(msg.AccountNonce, msg.Quantity, msg.Interest, msg.Term, msg.RelayerAddress, msg.UserAddress, msg.LendingToken, msg.CollateralToken, msg.AutoTopUp, msg.Status, msg.Side, msg.Type, msg.Hash, lendingId, tradeId, msg.ExtraData)
	signedTx, err := types.LendingSignTx(tx, types.LendingTxSigner{}, privateKey)
	if err != nil {
		log.Print(err)
	}
	fmt.Println("nonce", nonce, "side", msg.Side, "quantity", new(big.Int).Div(msg.Quantity, _1E8), "Interest", new(big.Int).Div(new(big.Int).SetUint64(msg.Interest), _1E8), "%")

	err = client.SendLendingTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
	}
}

func TestSendLending(t *testing.T) {
	t.SkipNow() //TODO: remove it to run this test
	key := ""
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Print(err)
	}
	nonce, err := getLendingNonce(crypto.PubkeyToAddress(privateKey.PublicKey))
	if err != nil {
		t.Error("fail to get nonce")
		t.FailNow()
	}

	for true {
		// 10%
		interestRate := 10 * common.BaseLendingInterest.Uint64()
		// lendToken: USD, collateral: BTC
		// amount 1000 USD
		testSendLending(key, nonce, USDAddress, common.Address{}, new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
		testSendLending(key, nonce, USDAddress, BTCAddress, new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)

		// lendToken: USD, collateral: XDC
		// amount 1000 USD
		testSendLending(key, nonce, USDAddress, common.Address{}, new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
		testSendLending(key, nonce, USDAddress, common.HexToAddress(common.XDCNativeAddress), new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)

		// lendToken: BTC, collateral: XDC
		// amount 1 BTC
		testSendLending(key, nonce, BTCAddress, common.Address{}, new(big.Int).Mul(_1E18, big.NewInt(1)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
		testSendLending(key, nonce, BTCAddress, common.HexToAddress(common.XDCNativeAddress), new(big.Int).Mul(_1E18, big.NewInt(1)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)

		// lendToken: BTC, collateral: ETH
		// amount 1 BTC
		testSendLending(key, nonce, BTCAddress, common.Address{}, new(big.Int).Mul(_1E18, big.NewInt(1)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
		testSendLending(key, nonce, BTCAddress, ETHAddress, new(big.Int).Mul(_1E18, big.NewInt(1)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)

		// lendToken: XDC, collateral: BTC
		// amount 1000 XDC
		testSendLending(key, nonce, common.HexToAddress(common.XDCNativeAddress), common.Address{}, new(big.Int).Mul(_1E18, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
		testSendLending(key, nonce, common.HexToAddress(common.XDCNativeAddress), BTCAddress, new(big.Int).Mul(_1E18, big.NewInt(1000)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)

		// lendToken: XDC, collateral: ETH
		// amount 1000 XDC
		testSendLending(key, nonce, common.HexToAddress(common.XDCNativeAddress), common.Address{}, new(big.Int).Mul(_1E18, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
		testSendLending(key, nonce, common.HexToAddress(common.XDCNativeAddress), ETHAddress, new(big.Int).Mul(_1E18, big.NewInt(1000)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
		nonce++
		time.Sleep(time.Second)
	}
}

func TestCancelLending(t *testing.T) {
	t.SkipNow() //TODO: remove it to run this test
	key := ""
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Print(err)
	}
	nonce, err := getLendingNonce(crypto.PubkeyToAddress(privateKey.PublicKey))
	if err != nil {
		t.Error("fail to get nonce")
		t.FailNow()
	}

	// 10%
	interestRate := 10 * common.BaseLendingInterest.Uint64()
	testSendLending(key, nonce, USDAddress, common.Address{}, new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
	nonce++
	time.Sleep(2 * time.Second)
	//TODO: run the above testcase first, then updating lendingId, Hash
	testSendLending(key, nonce, USDAddress, common.Address{}, new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusCancelled, true, 1, 0, common.HexToHash("0x3da4e24b9c0f60e04cdb4c4494de37203c6e1a354907cbd6d9bbbe2e52aecaab"), "")

}

func TestRecallLending(t *testing.T) {
	t.SkipNow() //TODO: remove it to run this test
	key := ""
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Print(err)
	}
	nonce, err := getLendingNonce(crypto.PubkeyToAddress(privateKey.PublicKey))
	if err != nil {
		t.Error("fail to get nonce")
		t.FailNow()
	}
	interestRate := 10 * common.BaseLendingInterest.Uint64()
	testSendLending(key, nonce, USDAddress, common.Address{}, new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Investing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
	time.Sleep(2 * time.Second)
	nonce, err = getLendingNonce(crypto.PubkeyToAddress(privateKey.PublicKey))
	if err != nil {
		t.Error("fail to get nonce")
		t.FailNow()
	}
	testSendLending(key, nonce, USDAddress, common.HexToAddress(common.XDCNativeAddress), new(big.Int).Mul(_1E8, big.NewInt(1000)), interestRate, lendingstate.Borrowing, lendingstate.LendingStatusNew, true, 0, 0, common.Hash{}, "")
	time.Sleep(2 * time.Second)
}
