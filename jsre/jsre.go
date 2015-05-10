package jsre

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/robertkrimen/otto"
)

/*
JSRE is a generic JS runtime environment embedding the otto JS interpreter.
It provides some helper functions to
- load code from files
- run code snippets
- require libraries
- bind native go objects
*/
type JSRE struct {
	assetPath string
	vm        *otto.Otto

	evalQueue     chan *evalReq
	stopEventLoop chan bool
	loopWg        sync.WaitGroup
}

// jsTimer is a single timer instance with a callback function
type jsTimer struct {
	timer    *time.Timer
	duration time.Duration
	interval bool
	call     otto.FunctionCall
}

// evalResult is a structure to store the result of any serialized vm execution
type evalResult struct {
	result otto.Value
	err    error
}

// evalReq is a serialized vm execution request put in evalQueue and processed by runEventLoop
type evalReq struct {
	fn   func(res *evalResult)
	done chan bool
	res  evalResult
}

// runtime must be stopped with Stop() after use and cannot be used after stopping
func New(assetPath string) *JSRE {
	re := &JSRE{
		assetPath: assetPath,
		vm:        otto.New(),
	}

	// load prettyprint func definition
	re.vm.Run(pp_js)
	re.vm.Set("loadScript", re.loadScript)

	re.evalQueue = make(chan *evalReq)
	re.stopEventLoop = make(chan bool)
	re.loopWg.Add(1)
	go re.runEventLoop()

	return re
}

// this function runs a piece of JS code either in a serialized way (when useEQ is true) or instantly, circumventing the evalQueue
func (self *JSRE) run(src interface{}, useEQ bool) (value otto.Value, err error) {
	if useEQ {
		done := make(chan bool)
		req := &evalReq{
			fn: func(res *evalResult) {
				res.result, res.err = self.vm.Run(src)
			},
			done: done,
		}
		self.evalQueue <- req
		<-done
		return req.res.result, req.res.err
	} else {
		return self.vm.Run(src)
	}
}

/*
This function runs the main event loop from a goroutine that is started
 when JSRE is created. Use Stop() before exiting to properly stop it.
The event loop processes vm access requests from the evalQueue in a
 serialized way and calls timer callback functions at the appropriate time.

Exported functions always access the vm through the event queue. You can
 call the functions of the otto vm directly to circumvent the queue. These
 functions should be used if and only if running a routine that was already
 called from JS through an RPC call.
*/
func (self *JSRE) runEventLoop() {
	registry := map[*jsTimer]*jsTimer{}
	ready := make(chan *jsTimer)

	newTimer := func(call otto.FunctionCall, interval bool) (*jsTimer, otto.Value) {

		delay, _ := call.Argument(1).ToInteger()
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

		value, err := call.Otto.ToValue(timer)
		if err != nil {
			panic(err)
		}

		return timer, value
	}

	setTimeout := func(call otto.FunctionCall) otto.Value {
		_, value := newTimer(call, false)
		return value
	}

	setInterval := func(call otto.FunctionCall) otto.Value {
		_, value := newTimer(call, true)
		return value
	}

	clearTimeout := func(call otto.FunctionCall) otto.Value {
		timer, _ := call.Argument(0).Export()
		if timer, ok := timer.(*jsTimer); ok {
			timer.timer.Stop()
			delete(registry, timer)
		}
		return otto.UndefinedValue()
	}
	self.vm.Set("setTimeout", setTimeout)
	self.vm.Set("setInterval", setInterval)
	self.vm.Set("clearTimeout", clearTimeout)
	self.vm.Set("clearInterval", clearTimeout)

	var waitForCallbacks bool

loop:
	for {
		select {
		case timer := <-ready:
			// execute callback, remove/reschedule the timer
			var arguments []interface{}
			if len(timer.call.ArgumentList) > 2 {
				tmp := timer.call.ArgumentList[2:]
				arguments = make([]interface{}, 2+len(tmp))
				for i, value := range tmp {
					arguments[i+2] = value
				}
			} else {
				arguments = make([]interface{}, 1)
			}
			arguments[0] = timer.call.ArgumentList[0]
			_, err := self.vm.Call(`Function.call.call`, nil, arguments...)

			if err != nil {
				fmt.Println("js error:", err, arguments)
			}
			if timer.interval {
				timer.timer.Reset(timer.duration)
			} else {
				delete(registry, timer)
				if waitForCallbacks && (len(registry) == 0) {
					break loop
				}
			}
		case evalReq := <-self.evalQueue:
			// run the code, send the result back
			evalReq.fn(&evalReq.res)
			close(evalReq.done)
			if waitForCallbacks && (len(registry) == 0) {
				break loop
			}
		case waitForCallbacks = <-self.stopEventLoop:
			if !waitForCallbacks || (len(registry) == 0) {
				break loop
			}
		}
	}

	for _, timer := range registry {
		timer.timer.Stop()
		delete(registry, timer)
	}

	self.loopWg.Done()
}

// stops the event loop before exit, optionally waits for all timers to expire
func (self *JSRE) Stop(waitForCallbacks bool) {
	self.stopEventLoop <- waitForCallbacks
	self.loopWg.Wait()
}

// Exec(file) loads and runs the contents of a file
// if a relative path is given, the jsre's assetPath is used
func (self *JSRE) Exec(file string) error {
	return self.exec(common.AbsolutePath(self.assetPath, file), true)
}

// circumvents the eval queue, see runEventLoop
func (self *JSRE) execWithoutEQ(file string) error {
	return self.exec(common.AbsolutePath(self.assetPath, file), false)
}

func (self *JSRE) exec(path string, useEQ bool) error {
	code, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = self.run(code, useEQ)
	return err
}

// assigns value v to a variable in the JS environment
func (self *JSRE) Bind(name string, v interface{}) (err error) {
	self.Set(name, v)
	return
}

// runs a piece of JS code
func (self *JSRE) Run(code string) (otto.Value, error) {
	return self.run(code, true)
}

// returns the value of a variable in the JS environment
func (self *JSRE) Get(ns string) (otto.Value, error) {
	done := make(chan bool)
	req := &evalReq{
		fn: func(res *evalResult) {
			res.result, res.err = self.vm.Get(ns)
		},
		done: done,
	}
	self.evalQueue <- req
	<-done
	return req.res.result, req.res.err
}

// assigns value v to a variable in the JS environment
func (self *JSRE) Set(ns string, v interface{}) error {
	done := make(chan bool)
	req := &evalReq{
		fn: func(res *evalResult) {
			res.err = self.vm.Set(ns, v)
		},
		done: done,
	}
	self.evalQueue <- req
	<-done
	return req.res.err
}

/*
Executes a JS script from inside the currently executing JS code.
Should only be called from inside an RPC routine.
*/
func (self *JSRE) loadScript(call otto.FunctionCall) otto.Value {
	file, err := call.Argument(0).ToString()
	if err != nil {
		return otto.FalseValue()
	}
	if err := self.execWithoutEQ(file); err != nil { // loadScript is only called from inside js
		fmt.Println("err:", err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

// uses the "prettyPrint" JS function to format a value
func (self *JSRE) PrettyPrint(v interface{}) (val otto.Value, err error) {
	var method otto.Value
	v, err = self.ToValue(v)
	if err != nil {
		return
	}
	method, err = self.vm.Get("prettyPrint")
	if err != nil {
		return
	}
	return method.Call(method, v)
}

// creates an otto value from a go type (serialized version)
func (self *JSRE) ToValue(v interface{}) (otto.Value, error) {
	done := make(chan bool)
	req := &evalReq{
		fn: func(res *evalResult) {
			res.result, res.err = self.vm.ToValue(v)
		},
		done: done,
	}
	self.evalQueue <- req
	<-done
	return req.res.result, req.res.err
}

// creates an otto value from a go type (non-serialized version)
func (self *JSRE) ToVal(v interface{}) otto.Value {

	result, err := self.vm.ToValue(v)
	if err != nil {
		fmt.Println("Value unknown:", err)
		return otto.UndefinedValue()
	}
	return result
}

// evaluates JS function and returns result in a pretty printed string format
func (self *JSRE) Eval(code string) (s string, err error) {
	var val otto.Value
	val, err = self.Run(code)
	if err != nil {
		return
	}
	val, err = self.PrettyPrint(val)
	if err != nil {
		return
	}
	return fmt.Sprintf("%v", val), nil
}

// compiles and then runs a piece of JS code
func (self *JSRE) Compile(fn string, src interface{}) error {
	script, err := self.vm.Compile(fn, src)
	if err != nil {
		return err
	}
	self.run(script, true)
	return nil
}
