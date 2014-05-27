package ethui

import (
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/go-qml/qml"
)

type DebuggerWindow struct {
	win    *qml.Window
	engine *qml.Engine
	lib    *UiLib
	Db     *Debugger
}

func NewDebuggerWindow(lib *UiLib) *DebuggerWindow {
	engine := qml.NewEngine()
	component, err := engine.LoadFile(lib.AssetPath("debugger/debugger.qml"))
	if err != nil {
		fmt.Println(err)

		return nil
	}

	win := component.CreateWindow(nil)
	db := &Debugger{win, make(chan bool), true}

	return &DebuggerWindow{engine: engine, win: win, lib: lib, Db: db}
}

func (self *DebuggerWindow) Show() {
	context := self.engine.Context()
	context.SetVar("dbg", self)

	go func() {
		self.win.Show()
		self.win.Wait()
	}()
}

func (self *DebuggerWindow) Debug(valueStr, gasStr, gasPriceStr, data string) {
	state := self.lib.eth.BlockChain().CurrentBlock.State()

	script, err := ethutil.Compile(data)
	if err != nil {
		ethutil.Config.Log.Debugln(err)

		return
	}

	dis := ethchain.Disassemble(script)
	self.lib.win.Root().Call("clearAsm")

	for _, str := range dis {
		self.win.Root().Call("setAsm", str)
	}
	// Contract addr as test address
	keyPair := ethutil.GetKeyRing().Get(0)
	callerTx := ethchain.NewContractCreationTx(ethutil.Big(valueStr), ethutil.Big(gasStr), ethutil.Big(gasPriceStr), script)
	callerTx.Sign(keyPair.PrivateKey)

	account := self.lib.eth.StateManager().TransState().GetAccount(keyPair.Address())
	contract := ethchain.MakeContract(callerTx, state)
	callerClosure := ethchain.NewClosure(account, contract, contract.Init(), state, ethutil.Big(gasStr), ethutil.Big(gasPriceStr))

	block := self.lib.eth.BlockChain().CurrentBlock
	vm := ethchain.NewVm(state, self.lib.eth.StateManager(), ethchain.RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
	})

	self.Db.done = false
	go func() {
		callerClosure.Call(vm, contract.Init(), self.Db.halting)

		state.Reset()

		self.Db.done = true
	}()
}

func (self *DebuggerWindow) Next() {
	self.Db.Next()
}

type Debugger struct {
	win  *qml.Window
	N    chan bool
	done bool
}

type storeVal struct {
	Key, Value string
}

func (d *Debugger) halting(pc int, op ethchain.OpCode, mem *ethchain.Memory, stack *ethchain.Stack, stateObject *ethchain.StateObject) {
	d.win.Root().Call("setInstruction", pc)
	d.win.Root().Call("clearMem")
	d.win.Root().Call("clearStack")
	d.win.Root().Call("clearStorage")

	addr := 0
	for i := 0; i+32 <= mem.Len(); i += 32 {
		d.win.Root().Call("setMem", memAddr{fmt.Sprintf("%03d", addr), fmt.Sprintf("% x", mem.Data()[i:i+32])})
		addr++
	}

	for _, val := range stack.Data() {
		d.win.Root().Call("setStack", val.String())
	}

	stateObject.State().EachStorage(func(key string, node *ethutil.Value) {
		d.win.Root().Call("setStorage", storeVal{fmt.Sprintf("% x", key), fmt.Sprintf("% x", node.Str())})
	})

out:
	for {
		select {
		case <-d.N:
			break out
		default:
		}
	}
}

func (d *Debugger) Next() {
	if !d.done {
		d.N <- true
	}
}
