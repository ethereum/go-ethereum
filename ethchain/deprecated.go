package ethchain

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

func (sm *StateManager) MakeStateObject(state *State, tx *Transaction) *StateObject {
	contract := MakeContract(tx, state)
	if contract != nil {
		state.states[string(tx.CreationAddress())] = contract.state

		return contract
	}

	return nil
}

func (sm *StateManager) EvalScript(state *State, script []byte, object *StateObject, tx *Transaction, block *Block) (ret []byte, gas *big.Int, err error) {
	account := state.GetAccount(tx.Sender())

	err = account.ConvertGas(tx.Gas, tx.GasPrice)
	if err != nil {
		ethutil.Config.Log.Debugln(err)
		return
	}

	closure := NewClosure(account, object, script, state, tx.Gas, tx.GasPrice)
	vm := NewVm(state, sm, RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
		Value:       tx.Value,
		//Price:       tx.GasPrice,
	})
	ret, gas, err = closure.Call(vm, tx.Data, nil)

	// Update the account (refunds)
	state.UpdateStateObject(account)
	state.UpdateStateObject(object)

	return
}

func (self *StateManager) ProcessTransaction(tx *Transaction, coinbase *StateObject, state *State, toContract bool) (gas *big.Int, err error) {
	fmt.Printf("state root before update %x\n", state.Root())
	defer func() {
		if r := recover(); r != nil {
			ethutil.Config.Log.Infoln(r)
			err = fmt.Errorf("%v", r)
		}
	}()

	gas = new(big.Int)
	addGas := func(g *big.Int) { gas.Add(gas, g) }
	addGas(GasTx)

	// Get the sender
	sender := state.GetAccount(tx.Sender())

	if sender.Nonce != tx.Nonce {
		err = NonceError(tx.Nonce, sender.Nonce)
		return
	}

	sender.Nonce += 1
	defer func() {
		//state.UpdateStateObject(sender)
		// Notify all subscribers
		self.Ethereum.Reactor().Post("newTx:post", tx)
	}()

	txTotalBytes := big.NewInt(int64(len(tx.Data)))
	//fmt.Println("txTotalBytes", txTotalBytes)
	//txTotalBytes.Div(txTotalBytes, ethutil.Big32)
	addGas(new(big.Int).Mul(txTotalBytes, GasData))

	rGas := new(big.Int).Set(gas)
	rGas.Mul(gas, tx.GasPrice)

	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	totAmount := new(big.Int).Add(tx.Value, rGas)
	if sender.Amount.Cmp(totAmount) < 0 {
		state.UpdateStateObject(sender)
		err = fmt.Errorf("[TXPL] Insufficient amount in sender's (%x) account", tx.Sender())
		return
	}

	coinbase.BuyGas(gas, tx.GasPrice)
	state.UpdateStateObject(coinbase)
	fmt.Printf("1. root %x\n", state.Root())

	// Get the receiver
	receiver := state.GetAccount(tx.Recipient)

	// Send Tx to self
	if bytes.Compare(tx.Recipient, tx.Sender()) == 0 {
		// Subtract the fee
		sender.SubAmount(rGas)
	} else {
		// Subtract the amount from the senders account
		sender.SubAmount(totAmount)
		state.UpdateStateObject(sender)
		fmt.Printf("3. root %x\n", state.Root())

		// Add the amount to receivers account which should conclude this transaction
		receiver.AddAmount(tx.Value)
		state.UpdateStateObject(receiver)
		fmt.Printf("2. root %x\n", state.Root())
	}

	ethutil.Config.Log.Infof("[TXPL] Processed Tx %x\n", tx.Hash())

	return
}

func (sm *StateManager) ApplyTransaction(coinbase []byte, state *State, block *Block, tx *Transaction) (totalGasUsed *big.Int, err error) {
	/*
		Applies transactions to the given state and creates new
		state objects where needed.

		If said objects needs to be created
		run the initialization script provided by the transaction and
		assume there's a return value. The return value will be set to
		the script section of the state object.
	*/
	var (
		addTotalGas = func(gas *big.Int) { totalGasUsed.Add(totalGasUsed, gas) }
		gas         = new(big.Int)
		script      []byte
	)
	totalGasUsed = big.NewInt(0)
	snapshot := state.Snapshot()

	ca := state.GetAccount(coinbase)
	// Apply the transaction to the current state
	gas, err = sm.ProcessTransaction(tx, ca, state, false)
	addTotalGas(gas)
	fmt.Println("gas used by tx", gas)

	if tx.CreatesContract() {
		if err == nil {
			// Create a new state object and the transaction
			// as it's data provider.
			contract := sm.MakeStateObject(state, tx)
			if contract != nil {
				fmt.Println(Disassemble(contract.Init()))
				// Evaluate the initialization script
				// and use the return value as the
				// script section for the state object.
				script, gas, err = sm.EvalScript(state, contract.Init(), contract, tx, block)
				fmt.Println("gas used by eval", gas)
				addTotalGas(gas)
				fmt.Println("total =", totalGasUsed)

				fmt.Println("script len =", len(script))

				if err != nil {
					err = fmt.Errorf("[STATE] Error during init script run %v", err)
					return
				}
				contract.script = script
				state.UpdateStateObject(contract)
			} else {
				err = fmt.Errorf("[STATE] Unable to create contract")
			}
		} else {
			err = fmt.Errorf("[STATE] contract creation tx: %v for sender %x", err, tx.Sender())
		}
	} else {
		// Find the state object at the "recipient" address. If
		// there's an object attempt to run the script.
		stateObject := state.GetStateObject(tx.Recipient)
		if err == nil && stateObject != nil && len(stateObject.Script()) > 0 {
			_, gas, err = sm.EvalScript(state, stateObject.Script(), stateObject, tx, block)
			addTotalGas(gas)
		}
	}

	parent := sm.bc.GetBlock(block.PrevHash)
	total := new(big.Int).Add(block.GasUsed, totalGasUsed)
	limit := block.CalcGasLimit(parent)
	if total.Cmp(limit) > 0 {
		state.Revert(snapshot)
		err = GasLimitError(total, limit)
	}

	return
}

// Apply transactions uses the transaction passed to it and applies them onto
// the current processing state.
func (sm *StateManager) ApplyTransactions(coinbase []byte, state *State, block *Block, txs []*Transaction) ([]*Receipt, []*Transaction) {
	// Process each transaction/contract
	var receipts []*Receipt
	var validTxs []*Transaction
	var ignoredTxs []*Transaction // Transactions which go over the gasLimit

	totalUsedGas := big.NewInt(0)

	for _, tx := range txs {
		usedGas, err := sm.ApplyTransaction(coinbase, state, block, tx)
		if err != nil {
			if IsNonceErr(err) {
				continue
			}
			if IsGasLimitErr(err) {
				ignoredTxs = append(ignoredTxs, tx)
				// We need to figure out if we want to do something with thse txes
				ethutil.Config.Log.Debugln("Gastlimit:", err)
				continue
			}

			ethutil.Config.Log.Infoln(err)
		}

		accumelative := new(big.Int).Set(totalUsedGas.Add(totalUsedGas, usedGas))
		receipt := &Receipt{tx, ethutil.CopyBytes(state.Root().([]byte)), accumelative}

		receipts = append(receipts, receipt)
		validTxs = append(validTxs, tx)
	}

	fmt.Println("################# MADE\n", receipts, "\n############################")

	// Update the total gas used for the block (to be mined)
	block.GasUsed = totalUsedGas

	return receipts, validTxs
}
