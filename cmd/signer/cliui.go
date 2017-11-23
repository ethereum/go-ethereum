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
	"github.com/ethereum/go-ethereum/common"
	"os"
)

type CommandlineUI struct {
}

func NewCommandlineUI() *CommandlineUI {
	return &CommandlineUI{}
}
func confirm() bool {
	fmt.Printf("Type 'Yes' to approve\n$>")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := scanner.Text()
	if answer == "Yes" {
		return true
	}
	return false
}
func showMetadata(metadata Metadata) {
	fmt.Printf("Request info: %v -> %v -> %v\n", metadata.remote, metadata.scheme, metadata.local)
}

func (ui *CommandlineUI) ApproveTx(request *SignTxRequest, metadata Metadata, ch chan ApprovalStatus) {

	fmt.Printf("--------- Transaction request-------------\n")
	fmt.Printf("to:    %v\n", request.transaction.To())
	fmt.Printf("from:  %v\n", request.from)
	fmt.Printf("value: %v\n", request.transaction.Value())
	fmt.Printf("data:  %v\n", common.Bytes2Hex(request.transaction.Data()))
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- ApprovalStatus{common.Hash{}, confirm(), ""}
}
func (ui *CommandlineUI) ApproveSignData(request *SignDataRequest, metadata Metadata, ch chan ApprovalStatus) {

	fmt.Printf("-------- Sign data request--------------\n")
	fmt.Printf("account:  %x\n", request.account.Address)
	fmt.Printf("message:  \n%v\n", request.message)
	fmt.Printf("raw data: \n%v\n", request.rawdata)
	fmt.Printf("message hash:  %v\n", request.hash)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- ApprovalStatus{common.Hash{}, confirm(), ""}
}
func (ui *CommandlineUI) ApproveExport(request *ExportRequest, metadata Metadata, ch chan ApprovalStatus) {
	fmt.Printf("-------- Export account request--------------\n")
	fmt.Printf("A request has been made to export the (encrypted) keyfile\n")
	fmt.Printf("Approving this operation means that the caller obtains the (encrypted) contents\n")
	fmt.Printf("\n")
	fmt.Printf("account:  %x\n", request.account.Address)
	fmt.Printf("keyfile:  \n%v\n", request.file)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- ApprovalStatus{common.Hash{}, confirm(), ""}
}
func (ui *CommandlineUI) ApproveImport(request *ImportRequest, metadata Metadata, ch chan ApprovalStatus) {
	fmt.Printf("-------- Export account request--------------\n")
	fmt.Printf("A request has been made to import an encrypted keyfile\n")
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	ch <- ApprovalStatus{common.Hash{}, confirm(), ""}
}
func (ui *CommandlineUI) ApproveListing(request *ListRequest, metadata Metadata, ch chan ListApproval) {

	fmt.Printf("-------- List account request--------------\n")
	fmt.Printf("A request has been made to list all accounts. \n")
	fmt.Printf("You can select which accounts the caller can see\n")
	for _, account := range request.accounts {
		fmt.Printf("\t[x] %v\n", account.Address.Hex())
	}
	fmt.Printf("-------------------------------------------\n")
	showMetadata(metadata)
	if confirm() {
		ch <- ListApproval{request.accounts}
	} else {
		ch <- ListApproval{nil}
	}
}
func (ui *CommandlineUI) ApproveNewAccount(requst *NewAccountRequest, metadata Metadata, ch chan bool) {
	fmt.Printf("-------- New account request--------------\n")
	fmt.Printf("A request has been made to create a new. \n")
	fmt.Printf("Approving this operation means that a new account is created,\n")
	fmt.Printf("and the address show to the caller\n")
	showMetadata(metadata)
	ch <- confirm()
}

func (ui *CommandlineUI) ShowError(message string) {
	//stdout is used by communication
	fmt.Printf("ERROR: %v", message)
}
func (ui *CommandlineUI) ShowInfo(message string) {
	//stdout is used by communication
	fmt.Printf("Info: %v", message)
}
