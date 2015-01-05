// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
	"gopkg.in/qml.v1"
)

type DebuggerWindow struct {
	win    *qml.Window
	engine *qml.Engine
	lib    *UiLib

	vm *vm.DebugVm
	Db *Debugger

	state *state.StateDB
}

func NewDebuggerWindow(lib *UiLib) *DebuggerWindow {
	engine := qml.NewEngine()
	component, err := engine.LoadFile(lib.AssetPath("debugger/debugger.qml"))
	if err != nil {
		fmt.Println(err)

		return nil
	}

	win := component.CreateWindow(nil)

	w := &DebuggerWindow{engine: engine, win: win, lib: lib, vm: &vm.DebugVm{}}
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

	dis := core.Disassemble(data)
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
	self.Stop()

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

	statedb := self.lib.eth.ChainManager().TransState()
	account := self.lib.eth.ChainManager().TransState().GetAccount(keyPair.Address())
	contract := statedb.NewStateObject([]byte{0})
	contract.SetCode(script)
	contract.SetBalance(value)

	self.SetAsm(script)

	block := self.lib.eth.ChainManager().CurrentBlock()

	env := utils.NewEnv(self.lib.eth.ChainManager(), statedb, block, account.Address(), value)

	self.Logf("callsize %d", len(script))
	go func() {
		ret, err := env.Call(account, contract.Address(), data, gas, gasPrice, ethutil.Big0)
		//ret, g, err := callerClosure.Call(evm, data)
		tot := new(big.Int).Mul(env.Gas, gasPrice)
		self.Logf("gas usage %v total price = %v (%v)", env.Gas, tot, ethutil.CurrencyToString(tot))
		if err != nil {
			self.Logln("exited with errors:", err)
		} else {
			if len(ret) > 0 {
				self.Logf("exited: % x", ret)
			} else {
				self.Logf("exited: nil")
			}
		}

		statedb.Reset()

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

func (self *DebuggerWindow) Stop() {
	if !self.Db.done {
		self.Db.Q <- true
	}
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

func (self *Debugger) BreakHook(pc int, op vm.OpCode, mem *vm.Memory, stack *vm.Stack, stateObject *state.StateObject) bool {
	self.main.Logln("break on instr:", pc)

	return self.halting(pc, op, mem, stack, stateObject)
}

func (self *Debugger) StepHook(pc int, op vm.OpCode, mem *vm.Memory, stack *vm.Stack, stateObject *state.StateObject) bool {
	return self.halting(pc, op, mem, stack, stateObject)
}

func (self *Debugger) SetCode(byteCode []byte) {
	self.main.SetAsm(byteCode)
}

func (self *Debugger) BreakPoints() []int64 {
	return self.breakPoints
}

func (d *Debugger) halting(pc int, op vm.OpCode, mem *vm.Memory, stack *vm.Stack, stateObject *state.StateObject) bool {
	d.win.Root().Call("setInstruction", pc)
	d.win.Root().Call("clearMem")
	d.win.Root().Call("clearStack")
	d.win.Root().Call("clearStorage")

	addr := 0
	for i := 0; i+16 <= mem.Len(); i += 16 {
		dat := mem.Data()[i : i+16]
		var str string

		for _, d := range dat {
			if unicode.IsGraphic(rune(d)) {
				str += string(d)
			} else {
				str += "?"
			}
		}

		d.win.Root().Call("setMem", memAddr{fmt.Sprintf("%03d", addr), fmt.Sprintf("%s  % x", str, dat)})
		addr += 16
	}

	for _, val := range stack.Data() {
		d.win.Root().Call("setStack", val.String())
	}

	it := stateObject.Trie().Iterator()
	for it.Next() {
		d.win.Root().Call("setStorage", storeVal{fmt.Sprintf("% x", it.Key), fmt.Sprintf("% x", it.Value)})

	}

	stackFrameAt := new(big.Int).SetBytes(mem.Get(0, 32))
	psize := mem.Len() - int(new(big.Int).SetBytes(mem.Get(0, 32)).Uint64())
	d.win.Root().ObjectByName("stackFrame").Set("text", fmt.Sprintf(`<b>stack ptr</b>: %v`, stackFrameAt))
	d.win.Root().ObjectByName("stackSize").Set("text", fmt.Sprintf(`<b>stack size</b>: %d`, psize))
	d.win.Root().ObjectByName("memSize").Set("text", fmt.Sprintf(`<b>mem size</b>: %v`, mem.Len()))

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
