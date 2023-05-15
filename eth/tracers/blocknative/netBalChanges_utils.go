package blocknative

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type NBCMethodError string

func (e NBCMethodError) Error() string {
	return fmt.Sprintf("Invalid NBC method: %s", string(e))
}

// Checks for the specified method given when asked for the tracer to do netbalchanges
func (t *txnOpCodeTracer) checkNBCArgs() error {
	if t.opts.NBCMethod == "" {
		return NBCMethodError("tracerConfig.nbcMethod cannot be empty")
	}
	switch t.opts.NBCMethod {
	case "events", "internalTransactions", "storageSlots":
		return nil
	default:
		return NBCMethodError(fmt.Sprintf("Unknown nbcMethod: %s", t.opts.NBCMethod))
	}
}

// LookupAccount fetches details of an account and adds it to the prestate
// if it doesn't exist there.
func (t *txnOpCodeTracer) lookupAccount(addr common.Address) {
	if _, ok := t.trace.NetBalChanges.Pre[addr]; ok {
		return
	}
	var storage map[common.Hash]common.Hash

	if t.opts.NBCMethod == "storageSlot" {
		storage = make(map[common.Hash]common.Hash)
	} else {
		storage = nil
	}

	t.trace.NetBalChanges.Pre[addr] = &account{
		Balance: t.env.StateDB.GetBalance(addr),
		Storage: storage,
	}
}

// LookupStorage fetches the requested storage slot and adds
// it to the prestate of the given contract. It assumes `lookupAccount`
// has been performed on the contract before.
func (t *txnOpCodeTracer) lookupStorage(addr common.Address, key common.Hash) {
	if _, ok := t.trace.NetBalChanges.Pre[addr].Storage[key]; ok {
		return
	}
	t.trace.NetBalChanges.Pre[addr].Storage[key] = t.env.StateDB.GetState(addr, key)
}

// Here we capture data for the top level calls
func (t *txnOpCodeTracer) captureStartNBC(from common.Address, to common.Address, gas uint64, value *big.Int) {
	// lookupAccount will only create a storage slot map pre if t.opts.NBCMethod is "storageSlot"
	t.lookupAccount(from)
	t.lookupAccount(to)
	t.lookupAccount(t.env.Context.Coinbase)

	// Update the to address
	// The recipient balance includes the value transferred.
	toBal := new(big.Int).Sub(t.trace.NetBalChanges.Pre[to].Balance, value)
	t.trace.NetBalChanges.Pre[to].Balance = toBal

	// Collect the gas usage
	// We need to re-add them to get the pre-tx balance.
	gasPrice := t.env.TxContext.GasPrice
	consumedGas := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(t.trace.NetBalChanges.InitialGas))

	// Update the from address
	fromBal := new(big.Int).Set(t.trace.NetBalChanges.Pre[from].Balance)
	fromBal.Add(fromBal, new(big.Int).Add(value, consumedGas))
	t.trace.NetBalChanges.Pre[from].Balance = fromBal
}

func (t *txnOpCodeTracer) captureStateNBC(op vm.OpCode, scope *vm.ScopeContext) {
	stack := scope.Stack
	stackData := stack.Data()
	stackLen := len(stackData)
	caller := scope.Contract.Address()
	switch {
	// Only take storage slot if we want to use it!
	case t.opts.NBCMethod == "storageSlot" && stackLen >= 1 && (op == vm.SLOAD || op == vm.SSTORE):
		slot := common.Hash(stackData[stackLen-1].Bytes32())
		t.lookupStorage(caller, slot)
	case stackLen >= 1 && (op == vm.EXTCODECOPY || op == vm.EXTCODEHASH || op == vm.EXTCODESIZE || op == vm.BALANCE):
		addr := common.Address(stackData[stackLen-1].Bytes20())
		t.lookupAccount(addr)
	case stackLen >= 5 && (op == vm.DELEGATECALL || op == vm.CALL || op == vm.STATICCALL || op == vm.CALLCODE):
		addr := common.Address(stackData[stackLen-2].Bytes20())
		t.lookupAccount(addr)
	}
}

// For "event" styled collection of netbalchanges, create them here out of the event logs
func (t *txnOpCodeTracer) captureEventNBC(err error) {
	// We iterate through the logs for known events
	for _, log := range t.env.StateDB.Logs() {

		if len(log.Topics) == 0 {
			continue
		}

		eventSignature := log.Topics[0].Hex()

		switch eventSignature {
		case transferEventHex:
			var transfer struct {
				From     common.Address
				To       common.Address
				Value    *big.Int
				Contract common.Address
			}
			transfer.From = common.HexToAddress(log.Topics[1].Hex())
			transfer.To = common.HexToAddress(log.Topics[2].Hex())
			transfer.Value = new(big.Int).SetBytes(log.Data)
			transfer.Contract = log.Address

			if err != nil {
				continue
			}

			// Make token change object
			tokenchange := &Tokenchanges{
				From:     common.HexToAddress(log.Topics[1].Hex()),
				To:       common.HexToAddress(log.Topics[2].Hex()),
				Asset:    new(big.Int).SetBytes(log.Data),
				Contract: log.Address,
			}

			t.trace.NetBalChanges.Tokens = append(t.trace.NetBalChanges.Tokens, *tokenchange)
		default:
			// We pass over this event hex signature!
		}
	}
}

func (t *txnOpCodeTracer) collateNBC() {

	// Iterate through the collected accounts touched by the transaction execution
	// Create a post account if something useful was modified
	for addr, state := range t.trace.NetBalChanges.Pre {
		var postAccount *account
		modified := false

		// Check if Eth balance has changed
		newBalance := t.env.StateDB.GetBalance(addr)
		if newBalance.Cmp(t.trace.NetBalChanges.Pre[addr].Balance) != 0 {
			modified = true
		}

		// Check if storage was updated (only if required)
		if t.opts.NBCMethod == "storageSlot" {
			postAccount, modified = t.processPostAccountStorage(newBalance, addr, state)
		} else {
			postAccount = &account{Balance: newBalance, Storage: nil}
		}

		// Only add this if we see something useful was modified
		if modified {
			t.trace.NetBalChanges.Post[addr] = postAccount
		} else {
			delete(t.trace.NetBalChanges.Pre, addr)
		}
	}

	// Go through the modified accounts and build the net balance changes for ETH
	t.processPostAccountEth()
}

func (t *txnOpCodeTracer) processPostAccountEth() {
	for addr, state := range t.trace.NetBalChanges.Post {
		// Add the balance and storage separately, as one may not be changed but another is.
		preState, preExists := t.trace.NetBalChanges.Pre[addr]

		// If the post bal exists, add it to the diff
		var weiAmount *big.Int
		var etherAmount *big.Float
		if preExists && preState != nil && state.Balance != nil {
			weiAmount = new(big.Int).Sub(state.Balance, preState.Balance)
			etherAmount = weiToEther(weiAmount)
		}

		diff := &valueChange{
			Eth:      etherAmount,
			EthInWei: weiAmount,
		}
		t.trace.NetBalChanges.Balances[addr] = diff
	}
}

func (t *txnOpCodeTracer) processPostAccountStorage(newBalance *big.Int, addr common.Address, state *account) (*account, bool) {
	postAccount := &account{Balance: newBalance, Storage: make(map[common.Hash]common.Hash)}
	modified := false

	for key, val := range state.Storage {
		// Don't include the empty slot
		if val == (common.Hash{}) {
			delete(t.trace.NetBalChanges.Pre[addr].Storage, key)
		}
		newVal := t.env.StateDB.GetState(addr, key)
		if val == newVal {
			// Omit unchanged slots
			delete(t.trace.NetBalChanges.Pre[addr].Storage, key)
		} else {
			modified = true
			if newVal != (common.Hash{}) {
				postAccount.Storage[key] = newVal
			}
		}
	}
	return postAccount, modified
}

// This function attempts to get transfer events from internal transaction calls to token contracts
func (t *txnOpCodeTracer) processNBCFromTxn(from common.Address, contract common.Address, input []byte) {
	if len(input) < 4 {
		// Invalid input data
		return
	}

	// Everything below is for a erc20 / erc721 transfer decode event
	// Todo: elaborate on this for other types of transfer methods
	// Method ID (4 bytes) + Recipient Address (32 bytes) + Amount (32 bytes)
	methodID := input[:4]
	if strings.ToLower(fmt.Sprintf("%x", methodID)) != "a9059cbb" {
		// Not an ERC20 transfer
		return
	}

	to := common.BytesToAddress(input[4:36])
	amount := new(big.Int).SetBytes(input[36:68])

	// Make token change object
	tokenchange := &Tokenchanges{
		From:     from,
		To:       to,
		Asset:    amount,
		Contract: contract,
	}

	t.trace.NetBalChanges.Tokens = append(t.trace.NetBalChanges.Tokens, *tokenchange)

}
