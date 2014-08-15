package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
)

type DebuggerWindow struct {
	win    *qml.Window
	engine *qml.Engine
	lib    *UiLib

	vm *ethvm.Vm
	Db *Debugger

	state *ethstate.State
}

func NewDebuggerWindow(lib *UiLib) *DebuggerWindow {
	engine := qml.NewEngine()
	component, err := engine.LoadFile(lib.AssetPath("debugger/debugger.qml"))
	if err != nil {
		fmt.Println(err)

		return nil
	}

	win := component.CreateWindow(nil)

	w := &DebuggerWindow{engine: engine, win: win, lib: lib, vm: &ethvm.Vm{}}
	w.Db = NewDebugger(w)

	return w
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

func (self *DebuggerWindow) SetAsm(data []byte) {
	self.win.Root().Call("clearAsm")

	dis := ethchain.Disassemble(data)
	for _, str := range dis {
		self.win.Root().Call("setAsm", str)
	}
}

func (self *DebuggerWindow) Compile(code string) {
	var err error
	script := ethutil.StringToByteFunc(code, func(s string) (ret []byte) {
		ret, err = ethutil.Compile(s, true)
		return
	})

	if err == nil {
		self.SetAsm(script)
	}
}

// Used by QML
func (self *DebuggerWindow) AutoComp(code string) {
	if self.Db.done {
		self.Compile(code)
	}
}

func (self *DebuggerWindow) ClearLog() {
	self.win.Root().Call("clearLog")
}

func (self *DebuggerWindow) Debug(valueStr, gasStr, gasPriceStr, scriptStr, dataStr string) {
	if !self.Db.done {
		self.Db.Q <- true
	}

	defer func() {
		if r := recover(); r != nil {
			self.Logf("compile FAULT: %v", r)
		}
	}()

	data := utils.FormatTransactionData(dataStr)

	var err error
	script := ethutil.StringToByteFunc(scriptStr, func(s string) (ret []byte) {
		ret, err = ethutil.Compile(s, false)
		return
	})

	if err != nil {
		self.Logln(err)

		return
	}

	var (
		gas      = ethutil.Big(gasStr)
		gasPrice = ethutil.Big(gasPriceStr)
		value    = ethutil.Big(valueStr)
		// Contract addr as test address
		keyPair = self.lib.eth.KeyManager().KeyPair()
	)

	state := self.lib.eth.StateManager().TransState()
	account := self.lib.eth.StateManager().TransState().GetAccount(keyPair.Address())
	contract := ethstate.NewStateObject([]byte{0})
	contract.Balance = value

	self.SetAsm(script)

	block := self.lib.eth.BlockChain().CurrentBlock

	callerClosure := ethvm.NewClosure(account, contract, script, gas, gasPrice)
	env := utils.NewEnv(state, block, account.Address(), value)
	vm := ethvm.New(env)
	vm.Verbose = true
	vm.Dbg = self.Db

	self.vm = vm
	self.Db.done = false
	self.Logf("callsize %d", len(script))
	go func() {
		ret, g, err := callerClosure.Call(vm, data)
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

func (self *DebuggerWindow) Continue() {
	self.vm.Stepping = false
	self.Next()
}

func (self *DebuggerWindow) ExecCommand(command string) {
	if len(command) > 0 {
		cmd := strings.Split(command, " ")
		switch cmd[0] {
		case "help":
			self.Logln("Debugger commands:")
			self.Logln("break, bp                 Set breakpoint on instruction")
			self.Logln("clear [log, break, bp]    Clears previous set sub-command(s)")
		case "break", "bp":
			if len(cmd) > 1 {
				lineNo, err := strconv.Atoi(cmd[1])
				if err != nil {
					self.Logln(err)
					break
				}
				self.Db.breakPoints = append(self.Db.breakPoints, int64(lineNo))
				self.Logf("break point set on instruction %d", lineNo)
			} else {
				self.Logf("'%s' requires line number", cmd[0])
			}
		case "clear":
			if len(cmd) > 1 {
				switch cmd[1] {
				case "break", "bp":
					self.Db.breakPoints = nil

					self.Logln("Breakpoints cleared")
				case "log":
					self.ClearLog()
				default:
					self.Logf("clear '%s' is not valid", cmd[1])
				}
			} else {
				self.Logln("'clear' requires sub command")
			}

		default:
			self.Logf("Unknown command %s", cmd[0])
		}
	}
}

type Debugger struct {
	N               chan bool
	Q               chan bool
	done, interrupt bool
	breakPoints     []int64
	main            *DebuggerWindow
	win             *qml.Window
}

func NewDebugger(main *DebuggerWindow) *Debugger {
	db := &Debugger{make(chan bool), make(chan bool), true, false, nil, main, main.win}

	return db
}

type storeVal struct {
	Key, Value string
}

func (self *Debugger) BreakHook(pc int, op ethvm.OpCode, mem *ethvm.Memory, stack *ethvm.Stack, stateObject *ethstate.StateObject) bool {
	self.main.Logln("break on instr:", pc)

	return self.halting(pc, op, mem, stack, stateObject)
}

func (self *Debugger) StepHook(pc int, op ethvm.OpCode, mem *ethvm.Memory, stack *ethvm.Stack, stateObject *ethstate.StateObject) bool {
	return self.halting(pc, op, mem, stack, stateObject)
}

func (self *Debugger) SetCode(byteCode []byte) {
	self.main.SetAsm(byteCode)
}

func (self *Debugger) BreakPoints() []int64 {
	return self.breakPoints
}

func (d *Debugger) halting(pc int, op ethvm.OpCode, mem *ethvm.Memory, stack *ethvm.Stack, stateObject *ethstate.StateObject) bool {
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

	stateObject.EachStorage(func(key string, node *ethutil.Value) {
		d.win.Root().Call("setStorage", storeVal{fmt.Sprintf("% x", key), fmt.Sprintf("% x", node.Str())})
	})

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
