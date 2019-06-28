// Copyright 2015 The go-ethereum Authors
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

// Package jsre provides execution environment for JavaScript.
package jsre

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/dop251/goja"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/jsre/deps"
)

var (
	BignumberJs = deps.MustAsset("bignumber.js")
	Web3Js      = deps.MustAsset("web3.js")
)

/*
JSRE is a generic JS runtime environment embedding the goja JS interpreter.
It provides some helper functions to
- load code from files
- run code snippets
- require libraries
- bind native go objects
*/
type JSRE struct {
	assetPath     string
	output        io.Writer
	evalQueue     chan *evalReq
	stopEventLoop chan bool
	closed        chan struct{}
	vm            *goja.Runtime
}

// jsTimer is a single timer instance with a callback function
type jsTimer struct {
	timer    *time.Timer
	duration time.Duration
	interval bool
	call     goja.FunctionCall
}

// evalReq is a serialized vm execution request processed by runEventLoop.
type evalReq struct {
	fn   func(vm *goja.Runtime)
	done chan bool
}

// runtime must be stopped with Stop() after use and cannot be used after stopping
func New(assetPath string, output io.Writer) *JSRE {
	re := &JSRE{
		assetPath:     assetPath,
		output:        output,
		closed:        make(chan struct{}),
		evalQueue:     make(chan *evalReq),
		stopEventLoop: make(chan bool),
	}
	go re.runEventLoop()
	re.Set("loadScript", re.loadScript)
	re.Set("inspect", re.prettyPrintJS)
	return re
}

// randomSource returns a pseudo random value generator.
func randomSource() *rand.Rand {
	bytes := make([]byte, 8)
	seed := time.Now().UnixNano()
	if _, err := crand.Read(bytes); err == nil {
		seed = int64(binary.LittleEndian.Uint64(bytes))
	}

	src := rand.NewSource(seed)
	return rand.New(src)
}

// This function runs the main event loop from a goroutine that is started
// when JSRE is created. Use Stop() before exiting to properly stop it.
// The event loop processes vm access requests from the evalQueue in a
// serialized way and calls timer callback functions at the appropriate time.

// Exported functions always access the vm through the event queue. You can
// call the functions of the goja vm directly to circumvent the queue. These
// functions should be used if and only if running a routine that was already
// called from JS through an RPC call.
func (re *JSRE) runEventLoop() {
	defer close(re.closed)

	vm := goja.New()
	r := randomSource()
	vm.SetRandSource(r.Float64)

	registry := map[*jsTimer]*jsTimer{}
	ready := make(chan *jsTimer)

	newTimer := func(call goja.FunctionCall, interval bool) (*jsTimer, goja.Value) {
		delay := call.Argument(1).ToInteger()
		if 0 >= delay {
			delay = 1
		}
		timer := &jsTimer{
			duration: time.Duration(delay) * time.Millisecond,
			call:     call,
			interval: interval,
		}
		registry[timer] = timer

		timer.timer = time.AfterFunc(timer.duration, func() {
			ready <- timer
		})

		return timer, re.vm.ToValue(timer)
	}

	setTimeout := func(call goja.FunctionCall) goja.Value {
		_, value := newTimer(call, false)
		return value
	}

	setInterval := func(call goja.FunctionCall) goja.Value {
		_, value := newTimer(call, true)
		return value
	}

	clearTimeout := func(call goja.FunctionCall) goja.Value {
		timer := call.Argument(0).Export()
		if timer, ok := timer.(*jsTimer); ok {
			timer.timer.Stop()
			delete(registry, timer)
		}
		return goja.Undefined()
	}
	vm.Set("_setTimeout", setTimeout)
	vm.Set("_setInterval", setInterval)
	vm.RunString(`var setTimeout = function(args) {
		if (arguments.length < 1) {
			throw TypeError("Failed to execute 'setTimeout': 1 argument required, but only 0 present.");
		}
		return _setTimeout.apply(this, arguments);
	}`)
	vm.RunString(`var setInterval = function(args) {
		if (arguments.length < 1) {
			throw TypeError("Failed to execute 'setInterval': 1 argument required, but only 0 present.");
		}
		return _setInterval.apply(this, arguments);
	}`)
	vm.Set("clearTimeout", clearTimeout)
	vm.Set("clearInterval", clearTimeout)

	var waitForCallbacks bool

loop:
	for {
		select {
		case timer := <-ready:
			// execute callback, remove/reschedule the timer
			var arguments []interface{}
			if len(timer.call.Arguments) > 2 {
				tmp := timer.call.Arguments[2:]
				arguments = make([]interface{}, 2+len(tmp))
				for i, value := range tmp {
					arguments[i+2] = value
				}
			} else {
				arguments = make([]interface{}, 1)
			}
			arguments[0] = timer.call.Arguments[0]
			call, isFunc := goja.AssertFunction(vm.Get(`Function.call.call`))
			if !isFunc {
				panic(vm.ToValue("js error: Function.call.call is not a function"))
			}
			call(goja.Null(), timer.call.Arguments...)

			_, inreg := registry[timer] // when clearInterval is called from within the callback don't reset it
			if timer.interval && inreg {
				timer.timer.Reset(timer.duration)
			} else {
				delete(registry, timer)
				if waitForCallbacks && (len(registry) == 0) {
					break loop
				}
			}
		case req := <-re.evalQueue:
			// run the code, send the result back
			req.fn(vm)
			close(req.done)
			if waitForCallbacks && (len(registry) == 0) {
				break loop
			}
		case waitForCallbacks = <-re.stopEventLoop:
			if !waitForCallbacks || (len(registry) == 0) {
				break loop
			}
		}
	}

	for _, timer := range registry {
		timer.timer.Stop()
		delete(registry, timer)
	}
}

// Do executes the given function on the JS event loop.
func (re *JSRE) Do(fn func(*goja.Runtime)) {
	done := make(chan bool)
	req := &evalReq{fn, done}
	re.evalQueue <- req
	<-done
}

// stops the event loop before exit, optionally waits for all timers to expire
func (re *JSRE) Stop(waitForCallbacks bool) {
	select {
	case <-re.closed:
	case re.stopEventLoop <- waitForCallbacks:
		<-re.closed
	}
}

// Exec(file) loads and runs the contents of a file
// if a relative path is given, the jsre's assetPath is used
func (re *JSRE) Exec(file string) error {
	code, err := ioutil.ReadFile(common.AbsolutePath(re.assetPath, file))
	if err != nil {
		return err
	}
	var script *goja.Program
	re.Do(func(vm *goja.Runtime) {
		script, err = goja.Compile(file, string(code), false)
		if err != nil {
			return
		}
		_, err = vm.RunProgram(script)
	})
	return err
}

// Bind assigns value v to a variable in the JS environment
// This method is deprecated, use Set.
func (re *JSRE) Bind(name string, v interface{}) error {
	return re.Set(name, v)
}

// Run runs a piece of JS code.
func (re *JSRE) Run(code string) (v goja.Value, err error) {
	re.Do(func(vm *goja.Runtime) { v, err = vm.RunString(code) })
	return v, err
}

// Get returns the value of a variable in the JS environment.
func (re *JSRE) Get(ns string) (v goja.Value) {
	re.Do(func(vm *goja.Runtime) { v = vm.Get(ns) })
	return v
}

// Set assigns value v to a variable in the JS environment.
func (re *JSRE) Set(ns string, v interface{}) (err error) {
	re.Do(func(vm *goja.Runtime) { vm.Set(ns, v) })
	return err
}

// loadScript executes a JS script from inside the currently executing JS code.
func (re *JSRE) loadScript(call goja.FunctionCall) goja.Value {
	file := call.Argument(0).ToString().String()
	file = common.AbsolutePath(re.assetPath, file)
	source, err := ioutil.ReadFile(file)
	if err != nil {
		// Panicking with a goja.Value arg will cause a JS exception
		// in the caller.
		panic(re.vm.ToValue(fmt.Sprintf("Could not read file %s: %v", file, err)))
	}
	value, err := compileAndRun(re.vm, file, string(source))
	if err != nil {
		panic(re.vm.ToValue(fmt.Sprintf("Error while compiling or running script: %v", err)))
	}
	return value
}

// Evaluate executes code and pretty prints the result to the specified output
// stream.
func (re *JSRE) Evaluate(code string, w io.Writer) error {
	var fail error

	re.Do(func(vm *goja.Runtime) {
		val, err := vm.RunString(code)
		if err != nil {
			prettyError(vm, err, w)
		} else {
			prettyPrint(vm, val, w)
		}
		fmt.Fprintln(w)
	})
	return fail
}

// Compile compiles and then runs a piece of JS code.
func (re *JSRE) Compile(filename string, src string) (err error) {
	re.Do(func(vm *goja.Runtime) { _, err = compileAndRun(vm, filename, src) })
	return err
}

func compileAndRun(vm *goja.Runtime, filename string, src string) (goja.Value, error) {
	script, err := goja.Compile(filename, src, false)
	if err != nil {
		return goja.Null(), err
	}
	return vm.RunProgram(script)
}
