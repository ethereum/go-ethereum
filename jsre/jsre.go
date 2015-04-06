package jsre

import (
	"fmt"
	"io/ioutil"

	"github.com/robertkrimen/otto"

	"github.com/ethereum/go-ethereum/common"
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
}

func New(assetPath string) *JSRE {
	re := &JSRE{
		assetPath,
		otto.New(),
	}

	// load prettyprint func definition
	re.vm.Run(pp_js)
	re.vm.Set("loadScript", re.loadScript)

	return re
}

// Exec(file) loads and runs the contents of a file
// if a relative path is given, the jsre's assetPath is used
func (self *JSRE) Exec(file string) error {
	return self.exec(common.AbsolutePath(self.assetPath, file))
}

func (self *JSRE) exec(path string) error {
	code, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = self.vm.Run(code)
	return err
}

func (self *JSRE) Bind(name string, v interface{}) (err error) {
	self.vm.Set(name, v)
	return
}

func (self *JSRE) Run(code string) (otto.Value, error) {
	return self.vm.Run(code)
}

func (self *JSRE) Get(ns string) (otto.Value, error) {
	return self.vm.Get(ns)
}

func (self *JSRE) Set(ns string, v interface{}) error {
	return self.vm.Set(ns, v)
}

func (self *JSRE) loadScript(call otto.FunctionCall) otto.Value {
	file, err := call.Argument(0).ToString()
	if err != nil {
		return otto.FalseValue()
	}
	if err := self.Exec(file); err != nil {
		fmt.Println("err:", err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (self *JSRE) PrettyPrint(v interface{}) (val otto.Value, err error) {
	var method otto.Value
	v, err = self.vm.ToValue(v)
	if err != nil {
		return
	}
	method, err = self.vm.Get("prettyPrint")
	if err != nil {
		return
	}
	return method.Call(method, v)
}

func (self *JSRE) ToVal(v interface{}) otto.Value {
	result, err := self.vm.ToValue(v)
	if err != nil {
		fmt.Println("Value unknown:", err)
		return otto.UndefinedValue()
	}
	return result
}

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

func (self *JSRE) Compile(fn string, src interface{}) error {
	script, err := self.vm.Compile(fn, src)
	if err != nil {
		return err
	}
	self.vm.Run(script)
	return nil
}
