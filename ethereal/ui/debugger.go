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
	go func() {
		self.win.Show()
		self.win.Wait()
	}()
}

func (self *DebuggerWindow) DebugTx(recipient, valueStr, gasStr, gasPriceStr, data string) {
	state := self.lib.eth.BlockChain().CurrentBlock.State()

	script, err := ethutil.Compile(data)
	if err != nil {
		ethutil.Config.Log.Debugln(err)

		return
	}

	dis := ethchain.Disassemble(script)
	self.lib.win.Root().Call("clearAsm")

	for _, str := range dis {
		self.lib.win.Root().Call("setAsm", str)
	}
	// Contract addr as test address
	keyPair := ethutil.GetKeyRing().Get(0)
	callerTx := ethchain.NewContractCreationTx(ethutil.Big(valueStr), ethutil.Big(gasStr), ethutil.Big(gasPriceStr), script)
	callerTx.Sign(keyPair.PrivateKey)

	account := self.lib.eth.StateManager().TransState().GetStateObject(keyPair.Address())
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
