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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

func (ui *CommandlineUI) RegisterUIServer(api *UIServerAPI) {
	// noop
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

func (ui *CommandlineUI) OnInputRequired(info UserInputRequest) (UserInputResponse, error) {

	fmt.Printf("## %s\n\n%s\n", info.Title, info.Prompt)
	if info.IsPassword {
		fmt.Printf("> ")
		text, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Error("Failed to read password", "err", err)
		}
		fmt.Println("-----------------------")
		return UserInputResponse{string(text)}, err
	}
	text := ui.readString()
	fmt.Println("-----------------------")
	return UserInputResponse{text}, nil
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
	fmt.Printf("\nAdditional HTTP header data, provided by the external caller:\n")
	fmt.Printf("\tUser-Agent: %v\n\tOrigin: %v\n", metadata.UserAgent, metadata.Origin)
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
	fmt.Printf("from:     %v\n", request.Transaction.From.String())
	fmt.Printf("value:    %v wei\n", weival)
	fmt.Printf("gas:      %v (%v)\n", request.Transaction.Gas, uint64(request.Transaction.Gas))
	fmt.Printf("gasprice: %v wei\n", request.Transaction.GasPrice.ToInt())
	fmt.Printf("nonce:    %v (%v)\n", request.Transaction.Nonce, uint64(request.Transaction.Nonce))
	if request.Transaction.Data != nil {
		d := *request.Transaction.Data
		if len(d) > 0 {

			fmt.Printf("data:     %v\n", hexutil.Encode(d))
		}
	}
	if request.Callinfo != nil {
		fmt.Printf("\nTransaction validation:\n")
		for _, m := range request.Callinfo {
			fmt.Printf("  * %s : %s\n", m.Typ, m.Message)
		}
		fmt.Println()

	}
	fmt.Printf("\n")
	showMetadata(request.Meta)
	fmt.Printf("-------------------------------------------\n")
	if !ui.confirm() {
		return SignTxResponse{request.Transaction, false}, nil
	}
	return SignTxResponse{request.Transaction, true}, nil
}

// ApproveSignData prompt the user for confirmation to request to sign data
func (ui *CommandlineUI) ApproveSignData(request *SignDataRequest) (SignDataResponse, error) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	fmt.Printf("-------- Sign data request--------------\n")
	fmt.Printf("Account:  %s\n", request.Address.String())
	fmt.Printf("messages:\n")
	for _, nvt := range request.Messages {
		fmt.Printf("\u00a0\u00a0%v\n", strings.TrimSpace(nvt.Pprint(1)))
	}
	fmt.Printf("raw data:  \n%q\n", request.Rawdata)
	fmt.Printf("data hash:  %v\n", request.Hash)
	fmt.Printf("-------------------------------------------\n")
	showMetadata(request.Meta)
	if !ui.confirm() {
		return SignDataResponse{false}, nil
	}
	return SignDataResponse{true}, nil
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
		fmt.Printf("  [x] %v\n", account.Address.Hex())
		fmt.Printf("    URL: %v\n", account.URL)
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

	fmt.Printf("-------- New Account request--------------\n\n")
	fmt.Printf("A request has been made to create a new account. \n")
	fmt.Printf("Approving this operation means that a new account is created,\n")
	fmt.Printf("and the address is returned to the external caller\n\n")
	showMetadata(request.Meta)
	if !ui.confirm() {
		return NewAccountResponse{false}, nil
	}
	return NewAccountResponse{true}, nil
}

// ShowError displays error message to user
func (ui *CommandlineUI) ShowError(message string) {
	fmt.Printf("## Error \n%s\n", message)
	fmt.Printf("-------------------------------------------\n")
}

// ShowInfo displays info message to user
func (ui *CommandlineUI) ShowInfo(message string) {
	fmt.Printf("## Info \n%s\n", message)
}

func (ui *CommandlineUI) OnApprovedTx(tx ethapi.SignTransactionResult) {
	fmt.Printf("Transaction signed:\n ")
	if jsn, err := json.MarshalIndent(tx.Tx, "  ", "  "); err != nil {
		fmt.Printf("WARN: marshalling error %v\n", err)
	} else {
		fmt.Println(string(jsn))
	}
}

func (ui *CommandlineUI) OnSignerStartup(info StartupInfo) {

	fmt.Printf("------- Signer info -------\n")
	for k, v := range info.Info {
		fmt.Printf("* %v : %v\n", k, v)
	}
}
