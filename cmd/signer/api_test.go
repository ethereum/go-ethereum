package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

//Used for testing
type HeadlessUI struct {
	controller chan string
}

func (ui *HeadlessUI) ApproveTx(request *SignTxRequest) (SignTxResponse, error) {

	switch <-ui.controller {
	case "Y":
		return SignTxResponse{request.Transaction, request.From, true, <-ui.controller}, nil
	case "M": //Modify
		old := (*big.Int)(request.Transaction.Value)
		newVal := big.NewInt(0).Add(old, big.NewInt(1))
		request.Transaction.Value = (*hexutil.Big)(newVal)
		return SignTxResponse{request.Transaction, request.From, true, <-ui.controller}, nil
	default:
		return SignTxResponse{request.Transaction, request.From, false, ""}, nil
	}
}
func (ui *HeadlessUI) ApproveSignData(request *SignDataRequest) (SignDataResponse, error) {
	if "Y" == <-ui.controller {
		return SignDataResponse{true, <-ui.controller}, nil
	}
	return SignDataResponse{false, ""}, nil
}
func (ui *HeadlessUI) ApproveExport(request *ExportRequest) (ExportResponse, error) {

	return ExportResponse{<-ui.controller == "Y"}, nil

}
func (ui *HeadlessUI) ApproveImport(request *ImportRequest) (ImportResponse, error) {

	if "Y" == <-ui.controller {
		return ImportResponse{true, <-ui.controller, <-ui.controller}, nil
	}
	return ImportResponse{false, "", ""}, nil
}
func (ui *HeadlessUI) ApproveListing(request *ListRequest) (ListResponse, error) {

	switch <-ui.controller {
	case "A":
		return ListResponse{request.Accounts}, nil
	case "1":
		l := make([]Account, 1)
		l[0] = request.Accounts[1]
		return ListResponse{l}, nil
	default:
		return ListResponse{nil}, nil
	}
}
func (ui *HeadlessUI) ApproveNewAccount(request *NewAccountRequest) (NewAccountResponse, error) {

	if "Y" == <-ui.controller {
		return NewAccountResponse{true, <-ui.controller}, nil
	}
	return NewAccountResponse{false, ""}, nil
}
func (ui *HeadlessUI) ShowError(message string) {
	//stdout is used by communication
	fmt.Fprint(os.Stderr, message)
}
func (ui *HeadlessUI) ShowInfo(message string) {
	//stdout is used by communication
	fmt.Fprint(os.Stderr, message)
}

func tmpDirName(t *testing.T) string {
	d, err := ioutil.TempDir("", "eth-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	d, err = filepath.EvalSymlinks(d)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func setup(t *testing.T) (*SignerAPI, chan string) {

	controller := make(chan string, 10)

	db, err := NewAbiDBFromFile(fmt.Sprintf("./4byte.json"))

	if err != nil {
		utils.Fatalf(err.Error())
	}
	var (
		ui  = &HeadlessUI{controller}
		api = NewSignerAPI(
			1,
			tmpDirName(t),
			true,
			ui,
			db,
			true)
	)
	return api, controller
}
func createAccount(control chan string, api *SignerAPI, t *testing.T) {

	control <- "Y"
	control <- "apassword"
	_, err := api.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Some time to allow changes to propagate
	time.Sleep(250 * time.Millisecond)
}
func failCreateAccount(control chan string, api *SignerAPI, t *testing.T) {
	control <- "N"
	acc, err := api.New(context.Background())
	if err != ErrRequestDenied {
		t.Fatal(err)
	}
	if acc.Address != (common.Address{}) {
		t.Fatal("Empty address should be returned")
	}
}
func list(control chan string, api *SignerAPI, t *testing.T) []Account {
	control <- "A"
	list, err := api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestNewAcc(t *testing.T) {

	api, control := setup(t)
	verifyNum := func(num int) {
		if list := list(control, api, t); len(list) != num {
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

	// Testing listing:
	// Listing one Account
	control <- "1"
	list, err := api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("List should only show one Account")
	}
	// Listing denied
	control <- "Nope"
	list, err = api.List(context.Background())
	if len(list) != 0 {
		t.Fatalf("List should be empty")
	}
	if err != ErrRequestDenied {
		t.Fatal("Expected deny")
	}
}

func TestSignData(t *testing.T) {

	api, control := setup(t)
	//Create two accounts
	createAccount(control, api, t)
	createAccount(control, api, t)
	control <- "1"
	list, err := api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := list[0].Address

	control <- "Y"
	control <- "wrongpassword"
	h, err := api.Sign(context.Background(), a, []byte("EHLO world"))
	if h != nil {
		t.Errorf("Expected nil-data, got %h", h)
	}
	if err != keystore.ErrDecrypt {
		t.Errorf("Expected ErrLocked! %v", err)
	}

	control <- "No way"
	h, err = api.Sign(context.Background(), a, []byte("EHLO world"))
	if h != nil {
		t.Errorf("Expected nil-data, got %h", h)
	}
	if err != ErrRequestDenied {
		t.Errorf("Expected ErrRequestDenied! %v", err)
	}

	control <- "Y"
	control <- "apassword"
	h, err = api.Sign(context.Background(), a, []byte("EHLO world"))

	if err != nil {
		t.Fatal(err)
	}
	if h == nil || len(h) != 65 {
		t.Errorf("Expected 65 byte signature (got %d bytes)", len(h))
	}
}
func mkTestTx() TransactionArg {
	to := common.HexToAddress("0x1337")
	gas := (*hexutil.Big)(big.NewInt(21000))
	gasPrice := (*hexutil.Big)(big.NewInt(2000000000))
	value := (*hexutil.Big)(big.NewInt(1e18))
	nonce := (hexutil.Uint64)(0)
	tx := TransactionArg{
		&to,
		gas,
		gasPrice,
		value,
		common.Hex2Bytes("01020304050607080a"),
		&nonce}
	return tx
}

func TestSignTx(t *testing.T) {

	var (
		list Accounts
		h    []byte
		h2   []byte
		err  error
	)

	api, control := setup(t)
	createAccount(control, api, t)
	control <- "A"
	list, err = api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := list[0].Address

	methodSig := "test(uint)"
	tx := mkTestTx()

	control <- "Y"
	control <- "wrongpassword"
	h, err = api.SignTransaction(context.Background(), a, tx, &methodSig)
	if h != nil {
		t.Errorf("Expected nil-data, got %h", h)
	}
	if err != keystore.ErrDecrypt {
		t.Errorf("Expected ErrLocked! %v", err)
	}

	control <- "No way"
	h, err = api.SignTransaction(context.Background(), a, tx, &methodSig)
	if h != nil {
		t.Errorf("Expected nil-data, got %h", h)
	}
	if err != ErrRequestDenied {
		t.Errorf("Expected ErrRequestDenied! %v", err)
	}

	control <- "Y"
	control <- "apassword"
	h, err = api.SignTransaction(context.Background(), a, tx, &methodSig)

	if err != nil {
		t.Fatal(err)
	}
	if h == nil || len(h) != 118 {
		t.Errorf("Expected 181 byte rlp-data (got %d bytes)", len(h))
	}
	//The tx is NOT modified by the UI
	control <- "Y"
	control <- "apassword"

	h2, err = api.SignTransaction(context.Background(), a, tx, &methodSig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(h, h2) {
		t.Error("Expected tx to be unmodified by UI")
	}

	//The tx is modified by the UI
	control <- "M"
	control <- "apassword"

	h2, err = api.SignTransaction(context.Background(), a, tx, &methodSig)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(h, h2) {
		t.Error("Expected tx to be modified by UI")
	}

}

/*
func TestAsyncronousResponses(t *testing.T){

	//Set up one account
	api, control := setup(t)
	createAccount(control, api, t)

	// Two transactions, the second one with larger value than the first
	tx1 := mkTestTx()
	newVal := big.NewInt(0).Add((*big.Int) (tx1.Value), big.NewInt(1))
	tx2 := mkTestTx()
	tx2.Value = (*hexutil.Big)(newVal)

	control <- "W" //wait
	control <- "Y" //
	control <- "apassword"
	control <- "Y" //
	control <- "apassword"

	var err error

	h1, err := api.SignTransaction(context.Background(), common.HexToAddress("1111"), tx1, nil)
	h2, err := api.SignTransaction(context.Background(), common.HexToAddress("2222"), tx2, nil)


	}
*/
