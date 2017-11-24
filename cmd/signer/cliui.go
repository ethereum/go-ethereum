// Copyright 2017 The go-ethereum Authors
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
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/ssh/terminal"
)

type CommandlineUI struct {
	in *bufio.Reader
}

func NewCommandlineUI() *CommandlineUI {
	return &CommandlineUI{bufio.NewReader(os.Stdin)}
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
	fmt.Printf("> ")
	text, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Crit("Failed to read password", "err", err)
	}
	fmt.Println()
	return string(text)
}

// confirm returns true if user enters 'Yes', otherwise false
func (ui *CommandlineUI) confirm() bool {
	fmt.Printf("Type 'Yes' to approve\n")
	if ui.readString() == "Yes" {
		return true
	}
	return false
}

func showMetadata(metadata Metadata) {
	fmt.Printf("Request info: %v -> %v -> %v\n", metadata.remote, metadata.scheme, metadata.local)
}

// ApproveTx prompt the user for confirmation to request to sign transaction
func (ui *CommandlineUI) ApproveTx(request *SignTxRequest, metadata Metadata, ch chan SignTxResponse) {

	fmt.Printf("--------- Transaction request-------------\n")
	fmt.Printf("to:    %v\n", request.transaction.To())
	fmt.Printf("from:  %v\n", request.from)
	fmt.Printf("value: %v\n", request.transaction.Value())
	fmt.Printf("data:  %v\n", common.Bytes2Hex(request.transaction.Data()))
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- SignTxResponse{request.transaction.Hash(), ui.confirm(), ""}
}

// ApproveSignData prompt the user for confirmation to request to sign data
func (ui *CommandlineUI) ApproveSignData(request *SignDataRequest, metadata Metadata, ch chan SignDataResponse) {

	fmt.Printf("-------- Sign data request--------------\n")
	fmt.Printf("account:  %x\n", request.account.Address)
	fmt.Printf("message:  \n%v\n", request.message)
	fmt.Printf("raw data: \n%v\n", request.rawdata)
	fmt.Printf("message hash:  %v\n", request.hash)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- SignDataResponse{ui.confirm(), ""}
}

// ApproveExport prompt the user for confirmation to export encrypted account json
func (ui *CommandlineUI) ApproveExport(request *ExportRequest, metadata Metadata, ch chan ExportResponse) {
	fmt.Printf("-------- Export account request--------------\n")
	fmt.Printf("A request has been made to export the (encrypted) keyfile\n")
	fmt.Printf("Approving this operation means that the caller obtains the (encrypted) contents\n")
	fmt.Printf("\n")
	fmt.Printf("account:  %x\n", request.account.Address)
	fmt.Printf("keyfile:  \n%v\n", request.file)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- ExportResponse{ui.confirm()}
}

// ApproveImport prompt the user for confirmation to import account json
func (ui *CommandlineUI) ApproveImport(request *ImportRequest, metadata Metadata, ch chan ImportResponse) {
	fmt.Printf("-------- Export account request--------------\n")
	fmt.Printf("A request has been made to import an encrypted keyfile\n")
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- ImportResponse{ui.confirm(), "", ""}
}

// ApproveListing prompt the user for confirmation to list accounts
// the list of accounts to list can be modified by the ui
func (ui *CommandlineUI) ApproveListing(request *ListRequest, metadata Metadata, ch chan ListResponse) {

	fmt.Printf("-------- List account request--------------\n")
	fmt.Printf("A request has been made to list all accounts. \n")
	fmt.Printf("You can select which accounts the caller can see\n")
	for _, account := range request.accounts {
		fmt.Printf("\t[x] %v\n", account.Address.Hex())
	}
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	if ui.confirm() {
		ch <- ListResponse{request.accounts}
	} else {
		ch <- ListResponse{nil}
	}
}

// ApproveNewAccount prompt the user for confirmation to create new account, and reveal to caller
func (ui *CommandlineUI) ApproveNewAccount(requst *NewAccountRequest, metadata Metadata, ch chan NewAccountResponse) {
	fmt.Printf("-------- New account request--------------\n")
	fmt.Printf("A request has been made to create a new. \n")
	fmt.Printf("Approving this operation means that a new account is created,\n")
	fmt.Printf("and the address show to the caller\n")
	showMetadata(metadata)
	ch <- NewAccountResponse{ui.confirm(), ""}
}

// ShowError displays error message to user
func (ui *CommandlineUI) ShowError(message string) {

	fmt.Printf("ERROR: %v", message)
}

// ShowInfo displays info message to user
func (ui *CommandlineUI) ShowInfo(message string) {

	fmt.Printf("Info: %v", message)
}
