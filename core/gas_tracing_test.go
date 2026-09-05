// Copyright 2026 The go-ethereum Authors
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

// These examples document the gas change event stream a transaction produces
// under EIP-8037, and pin it: the expected output is the contract.
//
// The stream is double-entry bookkeeping. Every transfer of gas between the
// transaction and a frame, or between a caller and a callee, is recorded on
// both budgets, so every budget opens at zero and closes at zero. Within a
// frame the execution and state dimensions are tracked separately; once the
// transaction settles they are one refundable amount, so the settlement events
// carry only the total. Charges made while computing an opcode's dynamic gas
// are covered by that opcode's own event and report nothing of their own.
//
// Runs of consecutive opcode events are collapsed to a single line showing
// their net movement, except that an opcode raising gas_left (an inline refill)
// is printed on its own. The chain property stays visible either way: each
// line starts where the previous line at the same depth ended.

package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// gasPrinter prints every gas event with the depth it belongs to.
type gasPrinter struct {
	depth   int
	opcodes int         // consecutive opcode events pending a summary line
	opFrom  tracing.Gas // where that run started
	opTo    tracing.Gas // where it currently ends
}

func gasStr(g tracing.Gas) string { return fmt.Sprintf("<%d,%d>", g.Execution, g.State) }

func (p *gasPrinter) flush() {
	if p.opcodes == 0 {
		return
	}
	fmt.Printf("d%d %-22s %s -> %s\n", p.depth, fmt.Sprintf("CallOpCode x%d", p.opcodes), gasStr(p.opFrom), gasStr(p.opTo))
	p.opcodes = 0
}

func (p *gasPrinter) hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnEnterV2: func(depth int, typ byte, from, to common.Address, input []byte, gas tracing.Gas, value *big.Int) {
			p.flush()
			fmt.Printf("d%d enter %s gas=%s\n", p.depth, vm.OpCode(typ), gasStr(gas))
			p.depth++
		},
		OnExitV2: func(depth int, output []byte, gasLeft tracing.Gas, err error, reverted bool) {
			p.flush()
			p.depth--
			fmt.Printf("d%d exit gasLeft=%s\n", p.depth, gasStr(gasLeft))
		},
		OnGasChangeV2: func(old, new tracing.Gas, reason tracing.GasChangeReason) {
			// Opcode events are collapsed into runs, except that one raising gas_left
			// (an inline refill) is printed on its own so the rise stays visible.
			if reason == tracing.GasChangeCallOpCode && new.Execution+new.State <= old.Execution+old.State {
				if p.opcodes == 0 {
					p.opFrom = old
				}
				p.opTo = new
				p.opcodes++
				return
			}
			p.flush()
			fmt.Printf("d%d %-22v %s -> %s\n", p.depth, reason, gasStr(old), gasStr(new))
		},
	}
}

// runTraced applies tx against alloc and prints its gas events.
func runTraced(alloc types.GenesisAlloc, tx *types.Transaction) {
	p := &gasPrinter{}
	evm := amsterdamTracedEVM(mkState(senderAlloc(alloc)), p.hooks())
	msg, err := TransactionToMessage(tx, signer8037, evm.Context.BaseFee)
	if err != nil {
		panic(err)
	}
	evm.SetTxContext(NewEVMTxContext(msg))
	res, err := newStateTransition(evm, msg, NewGasPool(evm.Context.GasLimit)).execute()
	if err != nil {
		panic(err)
	}
	p.flush()
	if res.Failed() {
		fmt.Printf("tx failed: %v\n", res.Err)
	}
}

var (
	traceCallee = common.BytesToAddress([]byte("callee"))
	traceChild  = common.BytesToAddress([]byte("child"))

	opStop    = []byte{0x00}
	opInvalid = []byte{0xfe}
	opRevert  = []byte{0x60, 0x00, 0x60, 0x00, 0xfd} // REVERT(0, 0)
	setSlot0  = []byte{0x60, 0x01, 0x60, 0x00, 0x55} // SSTORE(0, 1)
	clrSlot0  = []byte{0x60, 0x00, 0x60, 0x00, 0x55} // SSTORE(0, 0)
)

// callAll is bytecode that CALLs addr with all remaining gas and no value.
func callAll(addr common.Address) []byte {
	b := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x73}
	b = append(b, addr.Bytes()...)
	return append(b, 0x5a, 0xf1, 0x50) // GAS; CALL; POP
}

// callStipendOnly is bytecode that CALLs addr forwarding no gas at all but one
// wei of value, so the callee runs on nothing but the call stipend.
func callStipendOnly(addr common.Address) []byte {
	b := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x01, 0x73}
	b = append(b, addr.Bytes()...)
	return append(b, 0x60, 0x00, 0xf1, 0x50) // PUSH1 0 (gas); CALL; POP
}

// delegateAll is bytecode that DELEGATECALLs addr with all remaining gas, so
// the callee runs against this contract's storage.
func delegateAll(addr common.Address) []byte {
	b := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x73}
	b = append(b, addr.Bytes()...)
	return append(b, 0x5a, 0xf4, 0x50) // GAS; DELEGATECALL; POP
}

func concat(parts ...[]byte) (out []byte) {
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// The skeleton every transaction shares: the transaction hands its budget to
// the frame, the frame hands its leftover back, and both books close at zero.
func Example_plainCall() {
	runTraced(types.GenesisAlloc{traceCallee: {Code: opStop}}, callTx(0, traceCallee, 0, 100_000, nil))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <100000,0>
	// d0 TxIntrinsicGas         <100000,0> -> <85000,0>
	// d0 TxGasForwarded         <85000,0> -> <0,0>
	// d0 enter CALL gas=<85000,0>
	// d1 CallInitialBalance     <0,0> -> <85000,0>
	// d1 CallOpCode x1          <85000,0> -> <85000,0>
	// d1 CallLeftOverReturned   <85000,0> -> <0,0>
	// d0 exit gasLeft=<85000,0>
	// d0 CallLeftOverRefunded   <0,0> -> <85000,0>
	// d0 TxRefunds              <85000,0> -> <85000,0>
	// d0 TxLeftOverReturned     <85000,0> -> <0,0>
}

// Creating a slot with an empty reservoir spills the state charge into gas_left.
// Clearing it again refills gas_left: the opcode run ends higher than it began.
func Example_stateGasRefill() {
	runTraced(types.GenesisAlloc{traceCallee: {Code: concat(setSlot0, clrSlot0, opStop)}}, callTx(0, traceCallee, 0, 400_000, nil))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <400000,0>
	// d0 TxIntrinsicGas         <400000,0> -> <385000,0>
	// d0 TxGasForwarded         <385000,0> -> <0,0>
	// d0 enter CALL gas=<385000,0>
	// d1 CallInitialBalance     <0,0> -> <385000,0>
	// d1 CallOpCode x5          <385000,0> -> <274968,0>
	// d1 CallOpCode             <274968,0> -> <372788,0>
	// d1 CallOpCode x1          <372788,0> -> <372788,0>
	// d1 CallLeftOverReturned   <372788,0> -> <0,0>
	// d0 exit gasLeft=<372788,0>
	// d0 CallLeftOverRefunded   <0,0> -> <372788,0>
	// d0 TxRefunds              <372788,0> -> <378230,0>
	// d0 TxLeftOverReturned     <378230,0> -> <0,0>
}

// A reverting child is refilled the state gas of the creations it rolled back.
func Example_childRevert() {
	runTraced(types.GenesisAlloc{
		traceCallee: {Code: concat(callAll(traceChild), opStop)},
		traceChild:  {Code: concat(setSlot0, opRevert)},
	}, callTx(0, traceCallee, 0, 400_000, nil))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <400000,0>
	// d0 TxIntrinsicGas         <400000,0> -> <385000,0>
	// d0 TxGasForwarded         <385000,0> -> <0,0>
	// d0 enter CALL gas=<385000,0>
	// d1 CallInitialBalance     <0,0> -> <385000,0>
	// d1 CallOpCode x8          <385000,0> -> <5968,0>
	// d1 enter CALL gas=<376012,0>
	// d2 CallInitialBalance     <0,0> -> <376012,0>
	// d2 CallOpCode x6          <376012,0> -> <265980,0>
	// d2 RefundRevertedState    <265980,0> -> <363900,0>
	// d2 CallLeftOverReturned   <363900,0> -> <0,0>
	// d1 exit gasLeft=<363900,0>
	// d1 CallLeftOverRefunded   <5968,0> -> <369868,0>
	// d1 CallOpCode x2          <369868,0> -> <369866,0>
	// d1 CallLeftOverReturned   <369866,0> -> <0,0>
	// d0 exit gasLeft=<369866,0>
	// d0 CallLeftOverRefunded   <0,0> -> <369866,0>
	// d0 TxRefunds              <369866,0> -> <369866,0>
	// d0 TxLeftOverReturned     <369866,0> -> <0,0>
}

// A halting child burns its gas_left; with nothing left there is no leftover
// to hand back, so neither side of that transfer is recorded.
func Example_childHalt() {
	runTraced(types.GenesisAlloc{
		traceCallee: {Code: concat(callAll(traceChild), opStop)},
		traceChild:  {Code: concat(setSlot0, opInvalid)},
	}, callTx(0, traceCallee, 0, 400_000, nil))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <400000,0>
	// d0 TxIntrinsicGas         <400000,0> -> <385000,0>
	// d0 TxGasForwarded         <385000,0> -> <0,0>
	// d0 enter CALL gas=<385000,0>
	// d1 CallInitialBalance     <0,0> -> <385000,0>
	// d1 CallOpCode x8          <385000,0> -> <5968,0>
	// d1 enter CALL gas=<376012,0>
	// d2 CallInitialBalance     <0,0> -> <376012,0>
	// d2 CallOpCode x4          <376012,0> -> <265986,0>
	// d2 CallFailedExecution    <265986,0> -> <0,0>
	// d1 exit gasLeft=<0,0>
	// d1 CallOpCode x2          <5968,0> -> <5966,0>
	// d1 CallLeftOverReturned   <5966,0> -> <0,0>
	// d0 exit gasLeft=<5966,0>
	// d0 CallLeftOverRefunded   <0,0> -> <5966,0>
	// d0 TxRefunds              <5966,0> -> <5966,0>
	// d0 TxLeftOverReturned     <5966,0> -> <0,0>
}

// A value transfer hands the callee the 2300 gas stipend on top of whatever the
// caller forwards. The stipend is paid for by the caller's value-transfer fee,
// not moved from its budget, so this is the one transfer where what leaves the
// caller and what arrives at the callee differ. Whatever the callee does not
// use comes back to the caller like any other leftover.
func Example_valueCallStipend() {
	runTraced(types.GenesisAlloc{
		traceCallee: {Code: concat(callStipendOnly(traceChild), opStop), Balance: big.NewInt(1)},
		traceChild:  {Code: opStop},
	}, callTx(0, traceCallee, 0, 100_000, nil))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <100000,0>
	// d0 TxIntrinsicGas         <100000,0> -> <85000,0>
	// d0 TxGasForwarded         <85000,0> -> <0,0>
	// d0 enter CALL gas=<85000,0>
	// d1 CallInitialBalance     <0,0> -> <85000,0>
	// d1 CallOpCode x8          <85000,0> -> <70679,0>
	// d1 enter CALL gas=<2300,0>
	// d2 CallInitialBalance     <0,0> -> <2300,0>
	// d2 CallOpCode x1          <2300,0> -> <2300,0>
	// d2 CallLeftOverReturned   <2300,0> -> <0,0>
	// d1 exit gasLeft=<2300,0>
	// d1 CallLeftOverRefunded   <70679,0> -> <72979,0>
	// d1 CallOpCode x2          <72979,0> -> <72977,0>
	// d1 CallLeftOverReturned   <72977,0> -> <0,0>
	// d0 exit gasLeft=<72977,0>
	// d0 CallLeftOverRefunded   <0,0> -> <72977,0>
	// d0 TxRefunds              <72977,0> -> <72977,0>
	// d0 TxLeftOverReturned     <72977,0> -> <0,0>
}

// The caller creates a slot with an empty reservoir, spilling the state charge
// into its gas_left, and a DELEGATECALL'd child clears that slot. The refill
// lands in the child's reservoir, since the child owes nothing itself, and is
// handed back with the leftover. The merge then repays the caller's gas_left
// from it, as a separate movement.
func Example_crossFrameRefill() {
	runTraced(types.GenesisAlloc{
		traceCallee: {Code: concat(setSlot0, delegateAll(traceChild), opStop)},
		traceChild:  {Code: concat(clrSlot0, opStop)},
	}, callTx(0, traceCallee, 0, 400_000, nil))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <400000,0>
	// d0 TxIntrinsicGas         <400000,0> -> <385000,0>
	// d0 TxGasForwarded         <385000,0> -> <0,0>
	// d0 enter CALL gas=<385000,0>
	// d1 CallInitialBalance     <0,0> -> <385000,0>
	// d1 CallOpCode x10         <385000,0> -> <4249,0>
	// d1 enter DELEGATECALL gas=<267708,0>
	// d2 CallInitialBalance     <0,0> -> <267708,0>
	// d2 CallOpCode x2          <267708,0> -> <267702,0>
	// d2 CallOpCode             <267702,0> -> <267602,97920>
	// d2 CallOpCode x1          <267602,97920> -> <267602,97920>
	// d2 CallLeftOverReturned   <267602,97920> -> <0,0>
	// d1 exit gasLeft=<267602,97920>
	// d1 CallLeftOverRefunded   <4249,0> -> <271851,97920>
	// d1 StateGasRepaid         <271851,97920> -> <369771,0>
	// d1 CallOpCode x2          <369771,0> -> <369769,0>
	// d1 CallLeftOverReturned   <369769,0> -> <0,0>
	// d0 exit gasLeft=<369769,0>
	// d0 CallLeftOverRefunded   <0,0> -> <369769,0>
	// d0 TxRefunds              <369769,0> -> <375815,0>
	// d0 TxLeftOverReturned     <375815,0> -> <0,0>
}

// A contract creation: the transaction pays the account creation as a runtime
// charge, hands its gas to the init frame, and the init frame pays the code
// deposit in both dimensions.
func Example_create() {
	runTraced(nil, createTx(0, 400_000, deploy3))
	// Output:
	// d0 TxInitialBalance       <0,0> -> <400000,0>
	// d0 TxIntrinsicGas         <400000,0> -> <375930,0>
	// d0 TxRuntimeGas           <375930,0> -> <192330,0>
	// d0 TxGasForwarded         <192330,0> -> <0,0>
	// d0 enter CREATE gas=<192330,0>
	// d1 CallInitialBalance     <0,0> -> <192330,0>
	// d1 CallOpCode x3          <192330,0> -> <192321,0>
	// d1 CallCodeStorage        <192321,0> -> <192315,0>
	// d1 CallCodeStorage        <192315,0> -> <187725,0>
	// d1 CallLeftOverReturned   <187725,0> -> <0,0>
	// d0 exit gasLeft=<187725,0>
	// d0 CallLeftOverRefunded   <0,0> -> <187725,0>
	// d0 TxRefunds              <187725,0> -> <187725,0>
	// d0 TxLeftOverReturned     <187725,0> -> <0,0>
}

// A transaction that cannot cover a charge made before its first frame halts
// there. The gas above MaxTxGas forms a reservoir, which the synthetic frame
// hands back to the transaction like any halted frame would.
func Example_haltedBeforeFrame() {
	recipient := common.HexToAddress("0xc0ffee")
	tx := types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
		ChainID: cfg8037.ChainID, Nonce: 0, To: &recipient, Value: big.NewInt(1),
		Gas: params.MaxTxGas + 1000, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0),
		AccessList: types.AccessList{{Address: common.HexToAddress("0xdead"), StorageKeys: make([]common.Hash, 4094)}},
	})
	runTraced(nil, tx)
	// Output:
	// d0 TxInitialBalance       <0,0> -> <16778216,0>
	// d0 TxIntrinsicGas         <16778216,0> -> <179524,1000>
	// d0 TxGasForwarded         <179524,1000> -> <0,0>
	// d0 enter CALL gas=<179524,1000>
	// d1 CallInitialBalance     <0,0> -> <179524,1000>
	// d1 CallFailedExecution    <179524,1000> -> <0,1000>
	// d1 CallLeftOverReturned   <0,1000> -> <0,0>
	// d0 exit gasLeft=<0,1000>
	// d0 CallLeftOverRefunded   <0,0> -> <0,1000>
	// d0 TxRefunds              <1000,0> -> <1000,0>
	// d0 TxLeftOverReturned     <1000,0> -> <0,0>
	// tx failed: out of gas
}

// A runtime charge that succeeds before one that fails: the frame is shown
// entering with what the transaction had left after the successful charge.
func Example_authorizedThenHalted() {
	key, _ := crypto.HexToECDSA(authKeyA)
	auth, err := types.SignSetCode(key, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(cfg8037.ChainID), Address: delegate8037, Nonce: 0,
	})
	if err != nil {
		panic(err)
	}
	recipient := common.HexToAddress("0xc0ffee")
	tx := types.MustSignNewTx(senderKey, signer8037, &types.SetCodeTx{
		ChainID: uint256.MustFromBig(cfg8037.ChainID), Nonce: 0, To: recipient, Value: uint256.NewInt(1),
		Gas: 257_000, GasFeeCap: new(uint256.Int), GasTipCap: new(uint256.Int),
		AuthList: []types.SetCodeAuthorization{auth},
	})
	runTraced(nil, tx)
	// Output:
	// d0 TxInitialBalance       <0,0> -> <257000,0>
	// d0 TxIntrinsicGas         <257000,0> -> <228184,0>
	// d0 TxRuntimeGas           <228184,0> -> <394,0>
	// d0 TxGasForwarded         <394,0> -> <0,0>
	// d0 enter CALL gas=<394,0>
	// d1 CallInitialBalance     <0,0> -> <394,0>
	// d1 CallFailedExecution    <394,0> -> <0,0>
	// d0 exit gasLeft=<0,0>
	// d0 TxRefunds              <0,0> -> <0,0>
	// tx failed: out of gas
}
