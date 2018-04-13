// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/ssh/terminal"
)

type CommandlineUI struct {
	in *bufio.Reader
	mu sync.Mutex
}

func NewCommandlineUI() *CommandlineUI {
	return &CommandlineUI{in: bufio.NewReader(os.Stdin)}
}

// readString reads a single line from stdin, trimming if from spaces, enforcing
// non-emptyness.
func (ui *CommandlineUI) readString() string {
	for {
		fmt.Printf("> ")
		text, err := ui.in.ReadString('\n')
		if err != nil {
			log.Crit("Failed to read user input", "err", err)
		}
		if text = strings.TrimSpace(text); text != "" {
			return text
		}
	}
}

// readPassword reads a single line from stdin, trimming it from the trailing new
// line and returns it. The input will not be echoed.
func (ui *CommandlineUI) readPassword() string {
	fmt.Printf("Enter password to approve:\n")
	fmt.Printf("> ")

	text, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Crit("Failed to read password", "err", err)
	}
	fmt.Println()
	fmt.Println("-----------------------")
	return string(text)
}

// readPassword reads a single line from stdin, trimming it from the trailing new
// line and returns it. The input will not be echoed.
func (ui *CommandlineUI) readPasswordText(inputstring string) string {
	fmt.Printf("Enter %s:\n", inputstring)
	fmt.Printf("> ")
	text, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Crit("Failed to read password", "err", err)
	}
	fmt.Println("-----------------------")
	return string(text)
}

// confirm returns true if user enters 'Yes', otherwise false
func (ui *CommandlineUI) confirm() bool {
	fmt.Printf("Approve? [y/N]:\n")
	if ui.readString() == "y" {
		return true
	}
	fmt.Println("-----------------------")
	return false
}

func showMetadata(metadata Metadata) {
	fmt.Printf("Request context:\n\t%v -> %v -> %v\n", metadata.Remote, metadata.Scheme, metadata.Local)
}

// ApproveTx prompt the user for confirmation to request to sign Transaction
func (ui *CommandlineUI) ApproveTx(request *SignTxRequest) (SignTxResponse, error) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	weival := request.Transaction.Value.ToInt()
	fmt.Printf("--------- Transaction request-------------\n")
	if to := request.Transaction.To; to != nil {
		fmt.Printf("to:    %v\n", to.Original())
		if !to.ValidChecksum() {
			fmt.Printf("\nWARNING: Invalid checksum on to-address!\n\n")
		}
	} else {
		fmt.Printf("to:    <contact creation>\n")
	}
	fmt.Printf("from:  %v\n", request.Transaction.From.String())
	fmt.Printf("value: %v wei\n", weival)
	if request.Transaction.Data != nil {
		d := *request.Transaction.Data
		if len(d) > 0 {
			fmt.Printf("data:  %v\n", common.Bytes2Hex(d))
		}
	}
	if request.Callinfo != nil {
		fmt.Printf("\nTransaction validation:\n")
		for _, m := range request.Callinfo {
			fmt.Printf("  * %s : %s", m.Typ, m.Message)
		}
		fmt.Println()

	}
	fmt.Printf("\n")
	showMetadata(request.Meta)
	fmt.Printf("-------------------------------------------\n")
	if !ui.confirm() {
		return SignTxResponse{request.Transaction, false, ""}, nil
	}
	return SignTxResponse{request.Transaction, true, ui.readPassword()}, nil
}

// ApproveSignData prompt the user for confirmation to request to sign data
func (ui *CommandlineUI) ApproveSignData(request *SignDataRequest) (SignDataResponse, error) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	fmt.Printf("-------- Sign data request--------------\n")
	fmt.Printf("Account:  %s\n", request.Address.String())
	fmt.Printf("message:  \n%q\n", request.Message)
	fmt.Printf("raw data: \n%v\n", request.Rawdata)
	fmt.Printf("message hash:  %v\n", request.Hash)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(request.Meta)
	if !ui.confirm() {
		return SignDataResponse{false, ""}, nil
	}
	return SignDataResponse{true, ui.readPassword()}, nil
}

// ApproveExport prompt the user for confirmation to export encrypted Account json
func (ui *CommandlineUI) ApproveExport(request *ExportRequest) (ExportResponse, error) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	fmt.Printf("-------- Export Account request--------------\n")
	fmt.Printf("A request has been made to export the (encrypted) keyfile\n")
	fmt.Printf("Approving this operation means that the caller obtains the (encrypted) contents\n")
	fmt.Printf("\n")
	fmt.Printf("Account:  %x\n", request.Address)
	//fmt.Printf("keyfile:  \n%v\n", request.file)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(request.Meta)
	return ExportResponse{ui.confirm()}, nil
}

// ApproveImport prompt the user for confirmation to import Account json
func (ui *CommandlineUI) ApproveImport(request *ImportRequest) (ImportResponse, error) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	fmt.Printf("-------- Import Account request--------------\n")
	fmt.Printf("A request has been made to import an encrypted keyfile\n")
	fmt.Printf("-------------------------------------------\n")
	showMetadata(request.Meta)
	if !ui.confirm() {
		return ImportResponse{false, "", ""}, nil
	}
	return ImportResponse{true, ui.readPasswordText("Old password"), ui.readPasswordText("New password")}, nil
}

// ApproveListing prompt the user for confirmation to list accounts
// the list of accounts to list can be modified by the UI
func (ui *CommandlineUI) ApproveListing(request *ListRequest) (ListResponse, error) {

	ui.mu.Lock()
	defer ui.mu.Unlock()

	fmt.Printf("-------- List Account request--------------\n")
	fmt.Printf("A request has been made to list all accounts. \n")
	fmt.Printf("You can select which accounts the caller can see\n")
	for _, account := range request.Accounts {
		fmt.Printf("\t[x] %v\n", account.Address.Hex())
	}
	fmt.Printf("-------------------------------------------\n")
	showMetadata(request.Meta)
	if !ui.confirm() {
		return ListResponse{nil}, nil
	}
	return ListResponse{request.Accounts}, nil
}

// ApproveNewAccount prompt the user for confirmation to create new Account, and reveal to caller
func (ui *CommandlineUI) ApproveNewAccount(request *NewAccountRequest) (NewAccountResponse, error) {

	ui.mu.Lock()
	defer ui.mu.Unlock()

	fmt.Printf("-------- New Account request--------------\n")
	fmt.Printf("A request has been made to create a new. \n")
	fmt.Printf("Approving this operation means that a new Account is created,\n")
	fmt.Printf("and the address show to the caller\n")
	showMetadata(request.Meta)
	if !ui.confirm() {
		return NewAccountResponse{false, ""}, nil
	}
	return NewAccountResponse{true, ui.readPassword()}, nil
}

// ShowError displays error message to user
func (ui *CommandlineUI) ShowError(message string) {

	fmt.Printf("ERROR: %v\n", message)
}

// ShowInfo displays info message to user
func (ui *CommandlineUI) ShowInfo(message string) {
	fmt.Printf("Info: %v\n", message)
}

func (ui *CommandlineUI) OnApprovedTx(tx ethapi.SignTransactionResult) {
	fmt.Printf("Transaction signed:\n ")
	spew.Dump(tx.Tx)
}

func (ui *CommandlineUI) OnSignerStartup(info StartupInfo) {

	fmt.Printf("------- Signer info -------\n")
	for k, v := range info.Info {
		fmt.Printf("* %v : %v\n", k, v)
	}
}
