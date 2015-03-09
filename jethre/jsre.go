package javascript

import (
	"fmt"
	"github.com/obscuren/otto"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethutil"
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

func NewJSRE(assetPath string) *JSRE {
	re := &JSRE{
		assetPath,
		otto.New(),
	}

	// load prettyprint func definition
	re.vm.Run(pp_js)
	return re
}

func (self *JSRE) Load(file string) error {
	return self.load(ethutil.AbsolutePath(self.assetPath, file))
}

func (self *JSRE) load(path string) error {
	code, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = self.vm.Run(code)
	return err
}

func (self *JSRE) Bind(name string, v interface{}) (err error) {
	self.vm.Set(name, v)
	var t otto.Value
	t, err = self.vm.Get(name)
	if err != nil {
		return
	}
	o := t.Object()
	o.Set("require", self.require)
	return
}

func (self *JSRE) Run(code string) (otto.Value, error) {
	return self.vm.Run(code)
}

func (self *JSRE) Require(file string) error {
	if len(filepath.Ext(file)) == 0 {
		file += ".js"
	}

	fh, err := os.Open(file)
	if err != nil {
		return err
	}

	content, _ := ioutil.ReadAll(fh)
	self.Run("exports = {};(function() {" + string(content) + "})();")

	return nil
}

func (self *JSRE) require(call otto.FunctionCall) otto.Value {
	file, err := call.Argument(0).ToString()
	if err != nil {
		return otto.UndefinedValue()
	}
	if err := self.Require(file); err != nil {
		fmt.Println("err:", err)
		return otto.UndefinedValue()
	}

	t, _ := self.vm.Get("exports")

	return t
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

func (self *JSRE) toVal(v interface{}) otto.Value {
	result, err := self.vm.ToValue(v)
	if err != nil {
		fmt.Println("Value unknown:", err)
		return otto.UndefinedValue()
	}
	return result
}
