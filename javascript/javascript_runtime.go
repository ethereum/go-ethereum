package javascript

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
)

var jsrelogger = logger.NewLogger("JSRE")

type JSRE struct {
	Vm   *otto.Otto
	xeth *xeth.XEth

	objectCb map[string][]otto.Value
}

func (jsre *JSRE) LoadExtFile(path string) {
	result, err := ioutil.ReadFile(path)
	if err == nil {
		jsre.Vm.Run(result)
	} else {
		jsrelogger.Infoln("Could not load file:", path)
	}
}

func (jsre *JSRE) LoadIntFile(file string) {
	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	jsre.LoadExtFile(path.Join(assetPath, file))
}

func NewJSRE(xeth *xeth.XEth) *JSRE {
	re := &JSRE{
		otto.New(),
		xeth,
		make(map[string][]otto.Value),
	}

	// Init the JS lib
	re.Vm.Run(jsLib)

	// Load extra javascript files
	re.LoadIntFile("bignumber.min.js")

	re.Bind("eth", &JSEthereum{re.xeth, re.Vm})

	re.initStdFuncs()

	jsrelogger.Infoln("started")

	return re
}

func (self *JSRE) Bind(name string, v interface{}) {
	self.Vm.Set(name, v)
}

func (self *JSRE) Run(code string) (otto.Value, error) {
	return self.Vm.Run(code)
}

func (self *JSRE) initStdFuncs() {
	t, _ := self.Vm.Get("eth")
	eth := t.Object()
	eth.Set("require", self.require)
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

	t, _ := self.Vm.Get("exports")

	return t
}
