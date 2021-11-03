package lendingstate

import (
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestLendingItem_VerifyLendingSide(t *testing.T) {
	tests := []struct {
		name    string
		fields  *LendingItem
		wantErr bool
	}{
		{"wrong side", &LendingItem{Side: "GIVE"}, true},
		{"side: borrowing", &LendingItem{Side: Borrowing}, false},
		{"side: investing", &LendingItem{Side: Investing}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LendingItem{
				Side: tt.fields.Side,
			}
			if err := l.VerifyLendingSide(); (err != nil) != tt.wantErr {
				t.Errorf("VerifyLendingSide() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLendingItem_VerifyLendingInterest(t *testing.T) {
	tests := []struct {
		name    string
		fields  *LendingItem
		wantErr bool
	}{
		{"no interest information", &LendingItem{}, true},
		{"negative interest", &LendingItem{Interest: big.NewInt(-1)}, true},
		{"zero interest", &LendingItem{Interest: Zero}, true},
		{"positive interest", &LendingItem{Interest: big.NewInt(2)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LendingItem{
				Interest: tt.fields.Interest,
			}
			if err := l.VerifyLendingInterest(); (err != nil) != tt.wantErr {
				t.Errorf("VerifyLendingSide() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLendingItem_VerifyLendingQuantity(t *testing.T) {
	tests := []struct {
		name    string
		fields  *LendingItem
		wantErr bool
	}{
		{"no quantity information", &LendingItem{}, true},
		{"negative quantity", &LendingItem{Quantity: big.NewInt(-1)}, true},
		{"zero quantity", &LendingItem{Quantity: Zero}, true},
		{"positive quantity", &LendingItem{Quantity: big.NewInt(2)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LendingItem{
				Quantity: tt.fields.Quantity,
			}
			if err := l.VerifyLendingQuantity(); (err != nil) != tt.wantErr {
				t.Errorf("VerifyLendingQuantity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLendingItem_VerifyLendingType(t *testing.T) {
	tests := []struct {
		name    string
		fields  *LendingItem
		wantErr bool
	}{
		{"type: stop limit", &LendingItem{Type: "stop limit"}, true},
		{"type: take profit", &LendingItem{Type: "take profit"}, true},
		{"type: limit", &LendingItem{Type: Limit}, false},
		{"type: market", &LendingItem{Type: Market}, false},
		{"type: topup", &LendingItem{Type: TopUp}, false},
		{"type: repay", &LendingItem{Type: Repay}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LendingItem{
				Type: tt.fields.Type,
			}
			if err := l.VerifyLendingType(); (err != nil) != tt.wantErr {
				t.Errorf("VerifyLendingType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLendingItem_VerifyLendingStatus(t *testing.T) {
	tests := []struct {
		name    string
		fields  *LendingItem
		wantErr bool
	}{

		{"status: new", &LendingItem{Status: LendingStatusNew}, false},
		{"status: open", &LendingItem{Status: LendingStatusOpen}, true},
		{"status: partial_filled", &LendingItem{Status: LendingStatusPartialFilled}, true},
		{"status: filled", &LendingItem{Status: LendingStatusFilled}, true},
		{"status: cancelled", &LendingItem{Status: LendingStatusCancelled}, false},
		{"status: rejected", &LendingItem{Status: LendingStatusReject}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LendingItem{
				Status: tt.fields.Status,
			}
			if err := l.VerifyLendingStatus(); (err != nil) != tt.wantErr {
				t.Errorf("VerifyLendingStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func SetFee(statedb *state.StateDB, coinbase common.Address, feeRate *big.Int) {
	locRelayerState := state.GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locHash := common.BytesToHash(new(big.Int).Add(locRelayerState, LendingRelayerStructSlots["fee"]).Bytes())
	statedb.SetState(common.HexToAddress(common.LendingRegistrationSMC), locHash, common.BigToHash(feeRate))
}

func SetCollateralDetail(statedb *state.StateDB, token common.Address, depositRate *big.Int, liquidationRate *big.Int, price *big.Int) {
	collateralState := GetLocMappingAtKey(token.Hash(), CollateralMapSlot)
	locDepositRate := state.GetLocOfStructElement(collateralState, CollateralStructSlots["depositRate"])
	locLiquidationRate := state.GetLocOfStructElement(collateralState, CollateralStructSlots["liquidationRate"])
	locCollateralPrice := state.GetLocOfStructElement(collateralState, CollateralStructSlots["price"])
	statedb.SetState(common.HexToAddress(common.LendingRegistrationSMC), locDepositRate, common.BigToHash(depositRate))
	statedb.SetState(common.HexToAddress(common.LendingRegistrationSMC), locLiquidationRate, common.BigToHash(liquidationRate))
	statedb.SetState(common.HexToAddress(common.LendingRegistrationSMC), locCollateralPrice, common.BigToHash(price))
}

func TestVerifyBalance(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	relayer := common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e")
	uAddr := common.HexToAddress("0xDeE6238780f98c0ca2c2C28453149bEA49a3Abc9")
	lendingToken := common.HexToAddress("0xd9bb01454c85247B2ef35BB5BE57384cC275a8cf")    // USD
	collateralToken := common.HexToAddress("0x4d7eA2cE949216D6b120f3AA10164173615A2b6C") // BTC

	SetFee(statedb, relayer, big.NewInt(100))
	SetCollateralDetail(statedb, collateralToken, big.NewInt(150), big.NewInt(110), big.NewInt(8000)) // BTC price: 8k USD

	// have 10k USD
	statedb.GetOrNewStateObject(lendingToken)
	if err := SetTokenBalance(uAddr, EtherToWei(big.NewInt(10000)), lendingToken, statedb); err != nil {
		t.Error(err.Error())
	}

	// have 2 BTC
	statedb.GetOrNewStateObject(collateralToken)
	if err := SetTokenBalance(uAddr, EtherToWei(big.NewInt(2)), collateralToken, statedb); err != nil {
		t.Error(err.Error())
	}
	lendingdb := rawdb.NewMemoryDatabase()
	stateCache := NewDatabase(lendingdb)
	lendingstatedb, _ := New(EmptyRoot, stateCache)

	// insert lendingItem1 for testing cancel (side investing)
	lendingItem1 := LendingItem{
		Quantity:        EtherToWei(big.NewInt(11000000000)),
		Interest:        big.NewInt(10),
		Side:            Investing,
		Type:            Limit,
		LendingToken:    lendingToken,
		CollateralToken: collateralToken,
		FilledAmount:    nil,
		Status:          LendingStatusOpen,
		Relayer:         relayer,
		Term:            uint64(30),
		UserAddress:     uAddr,
		Signature:       nil,
		Hash:            common.Hash{},
		TxHash:          common.Hash{},
		Nonce:           nil,
		CreatedAt:       time.Time{},
		UpdatedAt:       time.Time{},
		LendingId:       uint64(1),
		ExtraData:       "",
	}
	lendingstatedb.InsertLendingItem(GetLendingOrderBookHash(lendingItem1.LendingToken, lendingItem1.Term), common.BigToHash(new(big.Int).SetUint64(lendingItem1.LendingId)), lendingItem1)

	// insert lendingItem2 for testing cancel (side borrowing)
	lendingItem2 := LendingItem{
		Quantity:        EtherToWei(big.NewInt(8000)),
		Interest:        big.NewInt(10),
		Side:            Borrowing,
		Type:            Limit,
		LendingToken:    lendingToken,
		CollateralToken: collateralToken,
		FilledAmount:    nil,
		Status:          LendingStatusOpen,
		Relayer:         relayer,
		Term:            uint64(30),
		UserAddress:     uAddr,
		Signature:       nil,
		Hash:            common.Hash{},
		TxHash:          common.Hash{},
		Nonce:           nil,
		CreatedAt:       time.Time{},
		UpdatedAt:       time.Time{},
		LendingId:       uint64(2),
		ExtraData:       "",
	}
	lendingstatedb.InsertLendingItem(GetLendingOrderBookHash(lendingItem2.LendingToken, lendingItem2.Term), common.BigToHash(new(big.Int).SetUint64(lendingItem2.LendingId)), lendingItem2)

	// insert lendingTrade for testing deposit (side: borrowing)
	lendingstatedb.InsertTradingItem(
		GetLendingOrderBookHash(lendingItem2.LendingToken, lendingItem2.Term),
		uint64(1),
		LendingTrade{
			TradeId:         uint64(1),
			CollateralToken: collateralToken,
			LendingToken:    lendingToken,
			Borrower:        uAddr,
			Amount:          EtherToWei(big.NewInt(8000)),
			LiquidationTime: uint64(time.Now().AddDate(0, 1, 0).UnixNano()),
		},
	)

	// make a big lendingTrade to test case: not enough balance to process payment
	lendingstatedb.InsertTradingItem(
		GetLendingOrderBookHash(lendingItem2.LendingToken, lendingItem2.Term),
		uint64(2),
		LendingTrade{
			TradeId:         uint64(2),
			CollateralToken: collateralToken,
			LendingToken:    lendingToken,
			Borrower:        uAddr,
			Amount:          EtherToWei(big.NewInt(20000)), // user have 10k USD, expect: fail
			LiquidationTime: uint64(time.Now().AddDate(0, 1, 0).UnixNano()),
		},
	)
	tests := []struct {
		name    string
		fields  *LendingItem
		wantErr bool
	}{
		{"Investor doesn't have enough balance. side: investing, quantity 11k USD",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Investing,
				Type:            Limit,
				Status:          LendingStatusNew,
				Quantity:        EtherToWei(big.NewInt(11000)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
			},
			true,
		},
		{"Investor has enough balance. side: investing, quantity 10k USD",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Investing,
				Type:            Limit,
				Status:          LendingStatusNew,
				Quantity:        EtherToWei(big.NewInt(10000)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
			},
			false,
		},
		{"Investor cancel lendingItem",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Investing,
				Type:            Limit,
				Status:          LendingStatusCancelled,
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				Term:            lendingItem1.Term,
				LendingId:       uint64(1),
			},
			true,
		},
		{"Invalid status",
			&LendingItem{
				Side:   Investing,
				Status: "wrong_status",
				Type:   Limit,
			},
			true,
		},
		// have 2BTC = 16k USD => max borrow = 16 / 1.5 = 10.66
		{"Borrower doesn't have enough balance. side: borrowing, quantity 12k USD",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Type:            Limit,
				Status:          LendingStatusNew,
				Quantity:        EtherToWei(big.NewInt(12000)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
			},
			true,
		},
		// have 2BTC = 16k USD => max borrow = 16 / 1.5 = 10.66
		{"Borrower has enough balance. side: borrowing, quantity 10k USD",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Type:            Limit,
				Status:          LendingStatusNew,
				Quantity:        EtherToWei(big.NewInt(10000)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
			},
			false,
		},
		{"Borrower has enough balance to pay cancel fee. side: borrowing",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Type:            Limit,
				Status:          LendingStatusCancelled,
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				Term:            lendingItem2.Term,
				LendingId:       uint64(2),
			},
			false,
		},
		{"Make a deposit to an empty LendingTrade.",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Status:          LendingStatusNew,
				Type:            TopUp,
				Quantity:        EtherToWei(big.NewInt(1)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				ExtraData:       common.BigToAddress(big.NewInt(0)).Hex(),
			},
			true,
		},
		// have 2BTC. make deposit 1 BTC
		{"Borrower has enough balance to make a deposit. side: borrowing",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Status:          LendingStatusNew,
				Type:            TopUp,
				Quantity:        EtherToWei(big.NewInt(1)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				Term:            uint64(30),
				LendingTradeId:  uint64(1),
			},
			false,
		},
		{"Make a payment to an empty LendingTrade.",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Status:          LendingStatusNew,
				Type:            Repay,
				Quantity:        EtherToWei(big.NewInt(1)),
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				LendingTradeId:  uint64(0),
			},
			true,
		},
		// have 10k USDT
		{"Borrower has enough balance to make a payment transaction. side: borrowing",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Status:          LendingStatusNew,
				Type:            Repay,
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				Term:            uint64(30),
				LendingTradeId:  uint64(1),
			},
			false,
		},
		// have 10k USDT
		{"Borrower doesn't haave enough balance to make a payment transaction. side: borrowing",
			&LendingItem{
				UserAddress:     uAddr,
				Relayer:         relayer,
				Side:            Borrowing,
				Status:          LendingStatusNew,
				Type:            Repay,
				LendingToken:    lendingToken,
				CollateralToken: collateralToken,
				Term:            uint64(30),
				LendingTradeId:  uint64(2),
			},
			true,
		},
		{"Invalid status",
			&LendingItem{
				Side:   Borrowing,
				Status: LendingStatusOpen,
			},
			true,
		},
		{"Invalid side",
			&LendingItem{
				Side: "abc",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyBalance(true,
				statedb,
				lendingstatedb,
				tt.fields.Type,
				tt.fields.Side,
				tt.fields.Status,
				tt.fields.UserAddress,
				tt.fields.Relayer,
				tt.fields.LendingToken,
				tt.fields.CollateralToken,
				tt.fields.Quantity,
				EtherToWei(big.NewInt(1)),
				EtherToWei(big.NewInt(1)),
				EtherToWei(big.NewInt(2)),    // XDC price: 0.5 USD => USD/XDC = 2
				EtherToWei(big.NewInt(8000)), // BTC = 8000 USD
				tt.fields.Term,
				tt.fields.LendingId,
				tt.fields.LendingTradeId,
			); (err != nil) != tt.wantErr {
				t.Errorf("VerifyBalance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type LendingOrderMsg struct {
	AccountNonce    uint64         `json:"nonce"    gencodec:"required"`
	Quantity        *big.Int       `json:"quantity,omitempty"`
	RelayerAddress  common.Address `json:"relayerAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	CollateralToken common.Address `json:"collateralToken,omitempty"`
	LendingToken    common.Address `json:"lendingToken,omitempty"`
	Interest        uint64         `json:"interest,omitempty"`
	Term            uint64         `json:"term,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	LendingID       uint64         `json:"lendingID,omitempty"`
	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

func Test_CreateOrder(t *testing.T) {
	t.SkipNow()
	for i := 0; i < 1; i++ {
		sendOrder(uint64(i))
		time.Sleep(time.Microsecond)
	}
}

func sendOrder(nonce uint64) {
	rpcClient, err := rpc.DialHTTP("http://localhost:8501")
	defer rpcClient.Close()
	if err != nil {
		fmt.Println("rpc.DialHTTP failed", "err", err)
		os.Exit(1)
	}
	rand.Seed(time.Now().UTC().UnixNano())
	item := &LendingOrderMsg{
		AccountNonce:    nonce,
		Quantity:        EtherToWei(big.NewInt(1000)),
		RelayerAddress:  common.HexToAddress("0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e"),
		UserAddress:     common.HexToAddress("0x17F2beD710ba50Ed27aEa52fc4bD7Bda5ED4a037"),
		CollateralToken: common.HexToAddress("0xC2fa1BA90b15E3612E0067A0020192938784D9C5"),
		LendingToken:    common.HexToAddress("0x45c25041b8e6CBD5c963E7943007187C3673C7c9"),
		Interest:        uint64(100),
		Term:            uint64(30 * 86400),
		Status:          LendingStatusNew,
		Side:            Borrowing,
		Type:            Limit,
		V:               common.Big0,
		R:               common.Big0,
		S:               common.Big0,
		Hash:            common.Hash{},
	}
	hash := computeHash(item)
	if item.Status != LendingStatusCancelled {
		item.Hash = hash
	}
	privKey, _ := crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	message := crypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		hash.Bytes(),
	)
	signatureBytes, _ := crypto.Sign(message, privKey)
	sig := &Signature{
		R: common.BytesToHash(signatureBytes[0:32]),
		S: common.BytesToHash(signatureBytes[32:64]),
		V: signatureBytes[64] + 27,
	}
	item.R = sig.R.Big()
	item.S = sig.S.Big()
	item.V = new(big.Int).SetUint64(uint64(sig.V))

	var result interface{}

	err = rpcClient.Call(&result, "XDCx_sendLending", item)
	fmt.Println("sendLendingitem", "nonce", item.AccountNonce)
	if err != nil {
		fmt.Println("rpcClient.Call XDCx_sendLending failed", "err", err)
		os.Exit(1)
	}
}

func computeHash(l *LendingOrderMsg) common.Hash {
	sha := sha3.NewKeccak256()
	if l.Status == LendingStatusCancelled {
		sha := sha3.NewKeccak256()
		sha.Write(l.Hash.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingID))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.RelayerAddress.Bytes())
	} else {
		sha.Write(l.RelayerAddress.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.CollateralToken.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(common.BigToHash(l.Quantity).Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
		if l.Type == Limit {
			sha.Write(common.BigToHash(big.NewInt(int64(l.Interest))).Bytes())
		}
		sha.Write([]byte(l.Side))
		sha.Write([]byte(l.Status))
		sha.Write([]byte(l.Type))
		sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
	}
	return common.BytesToHash(sha.Sum(nil))

}
