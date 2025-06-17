// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core_test

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/signer/core"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/ethereum/go-ethereum/signer/fourbyte"
	"github.com/ethereum/go-ethereum/signer/storage"
)

// Used for testing
type headlessUi struct {
	approveCh chan string // to send approve/deny
	inputCh   chan string // to send password
}

func (ui *headlessUi) OnInputRequired(info core.UserInputRequest) (core.UserInputResponse, error) {
	input := <-ui.inputCh
	return core.UserInputResponse{Text: input}, nil
}

func (ui *headlessUi) OnSignerStartup(info core.StartupInfo)        {}
func (ui *headlessUi) RegisterUIServer(api *core.UIServerAPI)       {}
func (ui *headlessUi) OnApprovedTx(tx ethapi.SignTransactionResult) {}

func (ui *headlessUi) ApproveTx(request *core.SignTxRequest) (core.SignTxResponse, error) {
	switch <-ui.approveCh {
	case "Y":
		return core.SignTxResponse{request.Transaction, true}, nil
	case "M": // modify
		// The headless UI always modifies the transaction
		old := big.Int(request.Transaction.Value)
		newVal := new(big.Int).Add(&old, big.NewInt(1))
		request.Transaction.Value = hexutil.Big(*newVal)
		return core.SignTxResponse{request.Transaction, true}, nil
	default:
		return core.SignTxResponse{request.Transaction, false}, nil
	}
}

func (ui *headlessUi) ApproveSignData(request *core.SignDataRequest) (core.SignDataResponse, error) {
	approved := (<-ui.approveCh == "Y")
	return core.SignDataResponse{approved}, nil
}

func (ui *headlessUi) ApproveListing(request *core.ListRequest) (core.ListResponse, error) {
	approval := <-ui.approveCh
	//fmt.Printf("approval %s\n", approval)
	switch approval {
	case "A":
		return core.ListResponse{request.Accounts}, nil
	case "1":
		l := make([]accounts.Account, 1)
		l[0] = request.Accounts[1]
		return core.ListResponse{l}, nil
	default:
		return core.ListResponse{nil}, nil
	}
}

func (ui *headlessUi) ApproveNewAccount(request *core.NewAccountRequest) (core.NewAccountResponse, error) {
	if <-ui.approveCh == "Y" {
		return core.NewAccountResponse{true}, nil
	}
	return core.NewAccountResponse{false}, nil
}

func (ui *headlessUi) ShowError(message string) {
	//stdout is used by communication
	fmt.Fprintln(os.Stderr, message)
}

func (ui *headlessUi) ShowInfo(message string) {
	//stdout is used by communication
	fmt.Fprintln(os.Stderr, message)
}

func tmpDirName(t *testing.T) string {
	d := t.TempDir()
	d, err := filepath.EvalSymlinks(d)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func setup(t *testing.T) (*core.SignerAPI, *headlessUi) {
	db, err := fourbyte.New()
	if err != nil {
		t.Fatal(err.Error())
	}
	ui := &headlessUi{make(chan string, 20), make(chan string, 20)}
	am := core.StartClefAccountManager(tmpDirName(t), true, true, "")
	api := core.NewSignerAPI(am, 1337, true, ui, db, true, &storage.NoStorage{})
	return api, ui
}
func createAccount(ui *headlessUi, api *core.SignerAPI, t *testing.T) {
	ui.approveCh <- "Y"
	ui.inputCh <- "a_long_password"
	_, err := api.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Some time to allow changes to propagate
	time.Sleep(250 * time.Millisecond)
}

func failCreateAccountWithPassword(ui *headlessUi, api *core.SignerAPI, password string, t *testing.T) {
	ui.approveCh <- "Y"
	// We will be asked three times to provide a suitable password
	ui.inputCh <- password
	ui.inputCh <- password
	ui.inputCh <- password

	addr, err := api.New(context.Background())
	if err == nil {
		t.Fatal("Should have returned an error")
	}
	if addr != (common.Address{}) {
		t.Fatal("Empty address should be returned")
	}
}

func failCreateAccount(ui *headlessUi, api *core.SignerAPI, t *testing.T) {
	ui.approveCh <- "N"
	addr, err := api.New(context.Background())
	if err != core.ErrRequestDenied {
		t.Fatal(err)
	}
	if addr != (common.Address{}) {
		t.Fatal("Empty address should be returned")
	}
}

func list(ui *headlessUi, api *core.SignerAPI, t *testing.T) ([]common.Address, error) {
	ui.approveCh <- "A"
	return api.List(context.Background())
}

func TestNewAcc(t *testing.T) {
	t.Parallel()
	api, control := setup(t)
	verifyNum := func(num int) {
		list, err := list(control, api, t)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if len(list) != num {
			t.Errorf("Expected %d accounts, got %d", num, len(list))
		}
	}
	// Testing create and create-deny
	createAccount(control, api, t)
	createAccount(control, api, t)
	failCreateAccount(control, api, t)
	failCreateAccount(control, api, t)
	createAccount(control, api, t)
	failCreateAccount(control, api, t)
	createAccount(control, api, t)
	failCreateAccount(control, api, t)
	verifyNum(4)

	// Fail to create this, due to bad password
	failCreateAccountWithPassword(control, api, "short", t)
	failCreateAccountWithPassword(control, api, "longerbutbad\rfoo", t)
	verifyNum(4)

	// Testing listing:
	// Listing one Account
	control.approveCh <- "1"
	list, err := api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("List should only show one Account")
	}
	// Listing denied
	control.approveCh <- "Nope"
	list, err = api.List(context.Background())
	if len(list) != 0 {
		t.Fatalf("List should be empty")
	}
	if err != core.ErrRequestDenied {
		t.Fatal("Expected deny")
	}
}

func mkTestTx(from common.MixedcaseAddress) apitypes.SendTxArgs {
	to := common.NewMixedcaseAddress(common.HexToAddress("0x1337"))
	gas := hexutil.Uint64(21000)
	gasPrice := (hexutil.Big)(*big.NewInt(2000000000))
	value := (hexutil.Big)(*big.NewInt(1e18))
	nonce := (hexutil.Uint64)(0)
	data := hexutil.Bytes(common.Hex2Bytes("01020304050607080a"))
	tx := apitypes.SendTxArgs{
		From:     from,
		To:       &to,
		Gas:      gas,
		GasPrice: &gasPrice,
		Value:    value,
		Data:     &data,
		Nonce:    nonce}
	return tx
}

func TestSignTx(t *testing.T) {
	t.Parallel()
	var (
		list      []common.Address
		res, res2 *ethapi.SignTransactionResult
		err       error
	)

	api, control := setup(t)
	createAccount(control, api, t)
	control.approveCh <- "A"
	list, err = api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("Unexpected empty list")
	}
	a := common.NewMixedcaseAddress(list[0])

	methodSig := "test(uint)"
	tx := mkTestTx(a)

	control.approveCh <- "Y"
	control.inputCh <- "wrongpassword"
	res, err = api.SignTransaction(context.Background(), tx, &methodSig)
	if res != nil {
		t.Errorf("Expected nil-response, got %v", res)
	}
	if err != keystore.ErrDecrypt {
		t.Errorf("Expected ErrLocked! %v", err)
	}
	control.approveCh <- "No way"
	res, err = api.SignTransaction(context.Background(), tx, &methodSig)
	if res != nil {
		t.Errorf("Expected nil-response, got %v", res)
	}
	if err != core.ErrRequestDenied {
		t.Errorf("Expected ErrRequestDenied! %v", err)
	}
	// Sign with correct password
	control.approveCh <- "Y"
	control.inputCh <- "a_long_password"
	res, err = api.SignTransaction(context.Background(), tx, &methodSig)

	if err != nil {
		t.Fatal(err)
	}
	parsedTx := &types.Transaction{}
	rlp.DecodeBytes(res.Raw, parsedTx)

	//The tx should NOT be modified by the UI
	if parsedTx.Value().Cmp(tx.Value.ToInt()) != 0 {
		t.Errorf("Expected value to be unchanged, expected %v got %v", tx.Value, parsedTx.Value())
	}
	control.approveCh <- "Y"
	control.inputCh <- "a_long_password"

	res2, err = api.SignTransaction(context.Background(), tx, &methodSig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res.Raw, res2.Raw) {
		t.Error("Expected tx to be unmodified by UI")
	}

	//The tx is modified by the UI
	control.approveCh <- "M"
	control.inputCh <- "a_long_password"

	res2, err = api.SignTransaction(context.Background(), tx, &methodSig)
	if err != nil {
		t.Fatal(err)
	}
	parsedTx2 := &types.Transaction{}
	rlp.DecodeBytes(res.Raw, parsedTx2)

	//The tx should be modified by the UI
	if parsedTx2.Value().Cmp(tx.Value.ToInt()) != 0 {
		t.Errorf("Expected value to be unchanged, got %v", parsedTx.Value())
	}
	if bytes.Equal(res.Raw, res2.Raw) {
		t.Error("Expected tx to be modified by UI")
	}
}

func ptr(s string) *string {
	return &s
}

func TestSIWE(t *testing.T) {
	tests := []struct {
		name      string
		request   core.SIWERequest
		signature string
		wantValid bool
	}{ 		
		{
			name: "example message",
			request: core.SIWERequest{
				Domain:         "login.xyz",
				Address:        "0x9D85ca56217D2bb651b00f15e694EB7E713637D4",
				Statement:      ptr("Sign-In With Ethereum Example Statement"),
				Uri:            "https://login.xyz",
				Version:        "1",
				Nonce:          "bTyXgcQxn2htgkjJn",
				IssuedAt:       "2022-01-27T17:09:38.578Z",
				ChainID:        1,
				ExpirationTime: ptr("2100-01-07T14:31:43.952Z"),
			},
			signature: "0xdc35c7f8ba2720df052e0092556456127f00f7707eaa8e3bbff7e56774e7f2e05a093cfc9e02964c33d86e8e066e221b7d153d27e5a2e97ccd5ca7d3f2ce06cb1b",
			wantValid: true,
		},
		{
			name: "recovery byte starting at 0",
			request: core.SIWERequest{
				Domain:    "www.tally.xyz",
				Address:   "0xc95EB884FE852e241D409234bfC7045CB9E31BD7",
				Statement: ptr("Sign in with Ethereum to Tally"),
				Uri:       "https://tally.xyz",
				Version:   "1",
				Nonce:     "15050747",
				IssuedAt:  "2022-06-30T14:08:51.382Z",
				ChainID:   1,
			},
			signature: "0x8c46b6eb8505939892d8e9b075f89f8277321b17b993151f37810cdda38cce6f4a85909d2b53e6a14629c74c0ac38bf4becde78ee5b2529812bf6cceaf7b2a2501",
			wantValid: true,
		},
		{
			name: "expired message",
			request: core.SIWERequest{
				Domain:    "login.xyz",
				Address:   "0x2ecA0068307e706741445764A3D6A4402aC2A5a9",
				Statement: ptr("Sign-In With Ethereum Example Statement"),
				Uri:       "https://login.xyz",
				Version:   "1",
				Nonce:     "lx2nx4so",
				IssuedAt:  "2022-01-05T14:27:30.883Z",
				ChainID:   1,
				ExpirationTime: ptr("2021-01-05T00:00:00Z"),
			},
			signature: "0x7337bc2826c7678cd6bc84f5b3b236efc969b0451f9feca2328b1d3401b030c113f19bdba359ba3f52762c66e9147311fa95fe598a1a4ec9bb383a7b4e3874241b",
			wantValid: false,
		},
		{
			name: "wrong signature",
			request: core.SIWERequest{
				Domain:    "login.xyz",
				Address:   "0x6Da01670d8fc844e736095918bbE11fE8D564163",
				Statement: ptr("Sign-In With Ethereum Example Statement"),
				Uri:       "https://login.xyz",
				Version:   "1",
				Nonce:     "rmplqh1gf",
				IssuedAt:  "2022-01-05T14:31:43.954Z",
				ChainID:   1,
				ExpirationTime: ptr("2100-01-07T14:31:43.952Z"),
			},
			signature: "0x31df81dc02344c9156e6f71da46e2db624b38f8f806290d670d46492b834b2e7575cbce9f48169356cfb577b910d8e30732fcf23c1ac0021d08b945ed7ee118e1b",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := tt.request.Verify(tt.signature)
			if valid != tt.wantValid {
				t.Errorf("expected valid=%v, got %v, err=%v", tt.wantValid, valid, err)
			}
		})
	}
}
