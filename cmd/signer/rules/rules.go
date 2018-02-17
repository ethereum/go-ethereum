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
	"github.com/ethereum/go-ethereum/cmd/signer/core"
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
	//	vm      *otto.Otto    // The JS vm
	next    core.SignerUI // The next handler, for manual processing
	storage storage.Storage
	jsRules string // The rules to use
}

func NewRuleEvaluator(next core.SignerUI) (*rulesetUi, error) {
	c := &rulesetUi{
		//		vm:      otto.New(),
		next:    next,
		storage: storage.NewEphemeralStorage(),
		jsRules: "",
	}

	return c, nil
}

func (r *rulesetUi) Init(javascriptRules string) error {
	r.jsRules = javascriptRules
	return nil
}
func (r *rulesetUi) execute(jsfunc string, jsarg interface{}) (otto.Value, error) {

	// Instantiate a fresh vm engine every time
	vm := otto.New()
	// Set the native callbacks
	consoleObj, _ := vm.Get("console")
	consoleObj.Object().Set("log", consoleOutput)
	consoleObj.Object().Set("error", consoleOutput)
	vm.Set("storage", r.storage)

	// Load bootstrap libraries
	script, err := vm.Compile("bignumber.js", BigNumber_JS)
	if err != nil {
		log.Warn("Failed loading libraries", "err", err)
		return otto.UndefinedValue(), err
	}
	vm.Run(script)

	// Run the actual rule implementation
	_, err = vm.Run(r.jsRules)
	if err != nil {
		log.Warn("Execution failed", "err", err)
		return otto.UndefinedValue(), err
	}

	// And the actual call
	// All calls are objects with the parameters being keys in that object.
	// To provide additional insulation between js and go, we serialize it into JSON on the Go-side,
	// and deserialize it on the JS side.
	//argdata := ""

	jsonbytes, err := json.Marshal(jsarg)
	if err != nil {
		log.Warn("failed marshalling data", "data", jsarg)
		return otto.UndefinedValue(), err
	}
	// Now, we call foobar(JSON.parse(<jsondata>)).
	var call string
	if len(jsonbytes) > 0 {
		call = fmt.Sprintf("%v(JSON.parse(%v))", jsfunc, string(jsonbytes))
	} else {
		call = fmt.Sprintf("%v()", jsfunc)
	}
	return vm.Run(call)
}

func (r *rulesetUi) checkApproval(jsfunc string, jsarg []byte, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	v, err := r.execute(jsfunc, string(jsarg))
	if err != nil {
		log.Info("error occurred during execution", "error", err)
		return false, err
	}
	result, err := v.ToString()
	if err != nil {
		log.Info("error occurred during response unmarshalling", "error", err)
		return false, err
	}
	if result == "Approve" {
		log.Info("Op approved")
		return true, nil
	} else if result == "Reject" {
		log.Info("Op rejected")
		return false, nil
	}
	return false, fmt.Errorf("Unknown response")
}

func (r *rulesetUi) ApproveTx(request *core.SignTxRequest) (core.SignTxResponse, error) {
	jsonreq, err := json.Marshal(request)
	approved, err := r.checkApproval("ApproveTx", jsonreq, err)
	if err != nil {
		log.Info("Rule-based approval error, going to manual", "error", "err")
		return r.next.ApproveTx(request)
	}
	if approved {
		return core.SignTxResponse{Transaction: request.Transaction, Approved: true, Password: ""}, nil
	}
	return core.SignTxResponse{Approved: false}, err
}

func (r *rulesetUi) ApproveSignData(request *core.SignDataRequest) (core.SignDataResponse, error) {
	jsonreq, err := json.Marshal(request)
	approved, err := r.checkApproval("ApproveSignData", jsonreq, err)
	if err != nil {
		log.Info("Rule-based approval error, going to manual", "error", "err")
		return r.next.ApproveSignData(request)
	}
	if approved {
		return core.SignDataResponse{Approved: true, Password: ""}, nil
	}
	return core.SignDataResponse{Approved: false, Password: ""}, err
}

func (r *rulesetUi) ApproveExport(request *core.ExportRequest) (core.ExportResponse, error) {
	jsonreq, err := json.Marshal(request)
	approved, err := r.checkApproval("ApproveExport", jsonreq, err)
	if err != nil {
		log.Info("Rule-based approval error, going to manual", "error", "err")
		return r.next.ApproveExport(request)
	}
	if approved {
		return core.ExportResponse{Approved: true}, nil
	}
	return core.ExportResponse{Approved: false}, err
}

func (r *rulesetUi) ApproveImport(request *core.ImportRequest) (core.ImportResponse, error) {
	// This cannot be handled by rules, requires setting a password
	// dispatch to next
	return r.next.ApproveImport(request)
}

func (r *rulesetUi) ApproveListing(request *core.ListRequest) (core.ListResponse, error) {
	jsonreq, err := json.Marshal(request)
	approved, err := r.checkApproval("ApproveListing", jsonreq, err)
	if err != nil {
		log.Info("Rule-based approval error, going to manual", "error", "err")
		return r.next.ApproveListing(request)
	}
	if approved {
		return core.ListResponse{Accounts: request.Accounts}, nil
	}
	return core.ListResponse{}, err
}

func (r *rulesetUi) ApproveNewAccount(request *core.NewAccountRequest) (core.NewAccountResponse, error) {
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
	_, err = r.execute("OnApprovedTx", string(jsonTx))
	if err != nil {
		log.Info("error occurred during execution", "error", err)
	}
}
