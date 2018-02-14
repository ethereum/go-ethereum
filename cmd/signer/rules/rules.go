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

package rules

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/cmd/signer"
	"github.com/ethereum/go-ethereum/cmd/signer/rules/deps"
	"github.com/ethereum/go-ethereum/cmd/signer/storage"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/robertkrimen/otto"
	"os"
	"strings"
)

var (
	BigNumber_JS = deps.MustAsset("bignumber.js")
)

// consoleOutput is an override for the console.log and console.error methods to
// stream the output into the configured output stream instead of stdout.
func consoleOutput(call otto.FunctionCall) otto.Value {
	output := []string{"JS:> "}
	for _, argument := range call.ArgumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	fmt.Fprintln(os.Stdout, strings.Join(output, " "))
	return otto.Value{}
}

// rulesetUi provides an implementation of SignerUI that evaluates a javascript
// file for each defined UI-method
type rulesetUi struct {
	vm      *otto.Otto      // The JS vm
	next    signer.SignerUI // The next handler, for manual processing
	storage storage.Storage
}

func NewRuleEvaluator() (*rulesetUi, error) {
	c := &rulesetUi{
		vm:      otto.New(),
		storage: storage.NewEphemeralStorage(),
	}
	consoleObj, _ := c.vm.Get("console")
	consoleObj.Object().Set("log", consoleOutput)
	consoleObj.Object().Set("error", consoleOutput)

	c.vm.Set("storage", c.storage)

	return c, nil
}

func (r *rulesetUi) Init(javascriptRules string) error {
	script, err := r.vm.Compile("bignumber.js", BigNumber_JS)
	if err != nil {
		log.Warn("Failed loading libraries", "err", err)
		return err
	}
	r.vm.Run(script)

	_, err = r.vm.Run(javascriptRules)
	if err != nil {
		log.Warn("Execution failed", "err", err)
	}
	return err
}

func (r *rulesetUi) checkApproval(jsfunc string, jsarg []byte, err error) error {
	if err != nil {
		return err
	}
	v, err := r.vm.Call(jsfunc, nil, string(jsarg))

	if err != nil {
		log.Info("error occurred during execution", "error", err)
		return err
	}
	result, err := v.ToString()
	if err != nil {
		log.Info("error occurred during response unmarshalling", "error", err)
		return err

	}
	if result == "Approve" {
		log.Info("Op approved")
		return nil
	}
	return fmt.Errorf("rejected")
}

func (r *rulesetUi) ApproveTx(request *signer.SignTxRequest) (signer.SignTxResponse, error) {
	jsonreq, err := json.Marshal(request)
	if err = r.checkApproval("ApproveTx", jsonreq, err); err == nil {
		return signer.SignTxResponse{Transaction: request.Transaction, Approved: true, Password: ""}, nil
	}
	return signer.SignTxResponse{Approved: false}, err
}

func (r *rulesetUi) ApproveSignData(request *signer.SignDataRequest) (signer.SignDataResponse, error) {
	jsonreq, err := json.Marshal(request)
	if err = r.checkApproval("ApproveTx", jsonreq, err); err == nil {
		return signer.SignDataResponse{Approved: true, Password: ""}, nil
	}
	return signer.SignDataResponse{Approved: false, Password: ""}, err
}

func (r *rulesetUi) ApproveExport(request *signer.ExportRequest) (signer.ExportResponse, error) {
	jsonreq, err := json.Marshal(request)
	if err = r.checkApproval("ApproveTx", jsonreq, err); err == nil {
		return signer.ExportResponse{Approved: true}, nil
	}
	return signer.ExportResponse{Approved: false}, err
}

func (r *rulesetUi) ApproveImport(request *signer.ImportRequest) (signer.ImportResponse, error) {
	// This cannot be handled by rules, requires setting a password
	// dispatch to next
	return r.next.ApproveImport(request)
}

func (r *rulesetUi) ApproveListing(request *signer.ListRequest) (signer.ListResponse, error) {
	jsonreq, err := json.Marshal(request)
	if err = r.checkApproval("ApproveListing", jsonreq, err); err == nil {
		return signer.ListResponse{Accounts: request.Accounts}, nil
	}
	return signer.ListResponse{}, err
}

func (r *rulesetUi) ApproveNewAccount(request *signer.NewAccountRequest) (signer.NewAccountResponse, error) {
	// This cannot be handled by rules, requires setting a password
	// dispatch to next
	return r.next.ApproveNewAccount(request)
}

func (r *rulesetUi) ShowError(message string) {
	log.Error(message)
	r.next.ShowError(message)
}

func (r *rulesetUi) ShowInfo(message string) {
	log.Info(message)
	r.next.ShowInfo(message)
}
func (r *rulesetUi) OnApprovedTx(tx ethapi.SignTransactionResult) {

	jsonTx, err := json.Marshal(tx)
	if err != nil {
		log.Warn("failed marshalling transaction", "tx", tx)
		return
	}
	_, err = r.vm.Call("OnApprovedTx", nil, string(jsonTx))
	if err != nil {
		fmt.Printf("Error in onapprove %v", err)
		log.Warn("error occurred during execution", "error", err)
	}

}
