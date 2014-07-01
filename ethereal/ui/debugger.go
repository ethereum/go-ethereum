package ethui

import (
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/go-qml/qml"
	"math/big"
	"strings"
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
	db := &Debugger{win, make(chan bool), make(chan bool), true, false, true}

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

func (self *DebuggerWindow) SetCode(code string) {
	self.win.Set("codeText", code)
}

func (self *DebuggerWindow) SetData(data string) {
	self.win.Set("dataText", data)
}
func (self *DebuggerWindow) SetAsm(data string) {
	dis := ethchain.Disassemble(ethutil.Hex2Bytes(data))
	for _, str := range dis {
		self.win.Root().Call("setAsm", str)
	}
}

func (self *DebuggerWindow) Debug(valueStr, gasStr, gasPriceStr, scriptStr, dataStr string) {
	if !self.Db.done {
		self.Db.Q <- true
	}
	self.Db.breakOnInstr = self.win.Root().ObjectByName("breakEachLine").Bool("checked")

	defer func() {
		if r := recover(); r != nil {
			self.Logf("compile FAULT: %v", r)
		}
	}()

	data := ethutil.StringToByteFunc(dataStr, func(s string) (ret []byte) {
		slice := strings.Split(dataStr, "\n")
		for _, dataItem := range slice {
			d := ethutil.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	var err error
	script := ethutil.StringToByteFunc(scriptStr, func(s string) (ret []byte) {
		ret, err = ethutil.Compile(s)
		return
	})

	if err != nil {
		self.Logln(err)

		return
	}

	dis := ethchain.Disassemble(script)
	self.win.Root().Call("clearAsm")
	self.win.Root().Call("clearLog")

	for _, str := range dis {
		self.win.Root().Call("setAsm", str)
	}

	var (
		gas      = ethutil.Big(gasStr)
		gasPrice = ethutil.Big(gasPriceStr)
		value    = ethutil.Big(valueStr)
		// Contract addr as test address
		keyPair  = self.lib.eth.KeyManager().KeyPair()
		callerTx = ethchain.NewContractCreationTx(ethutil.Big(valueStr), gas, gasPrice, script)
	)
	callerTx.Sign(keyPair.PrivateKey)

	state := self.lib.eth.BlockChain().CurrentBlock.State()
	account := self.lib.eth.StateManager().TransState().GetAccount(keyPair.Address())
	contract := ethchain.MakeContract(callerTx, state)
	contract.Amount = value
	callerClosure := ethchain.NewClosure(account, contract, script, state, gas, gasPrice)

	block := self.lib.eth.BlockChain().CurrentBlock
	vm := ethchain.NewVm(state, self.lib.eth.StateManager(), ethchain.RuntimeVars{
		Block:       block,
		Origin:      account.Address(),
		BlockNumber: block.Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
		Value:       ethutil.Big(valueStr),
	})
	vm.Verbose = true

	self.Db.done = false
	self.Logf("callsize %d", len(script))
	go func() {
		ret, g, err := callerClosure.Call(vm, data, self.Db.halting)
		tot := new(big.Int).Mul(g, gasPrice)
		self.Logf("gas usage %v total price = %v (%v)", g, tot, ethutil.CurrencyToString(tot))
		if err != nil {
			self.Logln("exited with errors:", err)
		} else {
			if len(ret) > 0 {
				self.Logf("exited: % x", ret)
			} else {
				self.Logf("exited: nil")
			}
		}

		state.Reset()

		if !self.Db.interrupt {
			self.Db.done = true
		} else {
			self.Db.interrupt = false
		}
	}()
}

func (self *DebuggerWindow) Logf(format string, v ...interface{}) {
	self.win.Root().Call("setLog", fmt.Sprintf(format, v...))
}

func (self *DebuggerWindow) Logln(v ...interface{}) {
	str := fmt.Sprintln(v...)
	self.Logf("%s", str[:len(str)-1])
}

func (self *DebuggerWindow) Next() {
	self.Db.Next()
}

type Debugger struct {
	win             *qml.Window
	N               chan bool
	Q               chan bool
	done, interrupt bool
	breakOnInstr    bool
}

type storeVal struct {
	Key, Value string
}

func (d *Debugger) halting(pc int, op ethchain.OpCode, mem *ethchain.Memory, stack *ethchain.Stack, stateObject *ethchain.StateObject) bool {
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

	if d.breakOnInstr {
	out:
		for {
			select {
			case <-d.N:
				break out
			case <-d.Q:
				d.interrupt = true
				d.clearBuffers()

				return false
			}
		}
	}

	return true
}

func (d *Debugger) clearBuffers() {
out:
	// drain
	for {
		select {
		case <-d.N:
		case <-d.Q:
		default:
			break out
		}
	}
}

func (d *Debugger) Next() {
	if !d.done {
		d.N <- true
	}
}
