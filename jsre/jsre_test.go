package jsre

import (
	"github.com/obscuren/otto"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/ethutil"
)

var defaultAssetPath = path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")

type testNativeObjectBinding struct {
	toVal func(interface{}) otto.Value
}

type msg struct {
	Msg string
}

func (no *testNativeObjectBinding) TestMethod(call otto.FunctionCall) otto.Value {
	m, err := call.Argument(0).ToString()
	if err != nil {
		return otto.UndefinedValue()
	}
	return no.toVal(&msg{m})
}

func TestExec(t *testing.T) {
	jsre := New("/tmp")

	ethutil.WriteFile("/tmp/test.js", []byte(`msg = "testMsg"`))
	err := jsre.Exec("test.js")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	val, err := jsre.Run("msg")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !val.IsString() {
		t.Errorf("expected string value, got %v", val)
	}

	// this errors
	err = jsre.Exec(path.Join(defaultAssetPath, "bignumber.min.js"))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	_, err = jsre.Run("x = new BigNumber(123.4567);")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBind(t *testing.T) {
	jsre := New(defaultAssetPath)

	jsre.Bind("no", &testNativeObjectBinding{jsre.ToVal})

	val, err := jsre.Run(`no.testMethod("testMsg")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	pp, err := jsre.PrettyPrint(val)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("no: %v", pp)
}

func TestRequire(t *testing.T) {
	jsre := New(defaultAssetPath)

	_, err := jsre.Run("x = new BigNumber(123.4567);")
	if err == nil {
		t.Errorf("expected error, got nothing")
	}
	_, err = jsre.Run(`loadScript("bignumber.min.js"); x = new BigNumber(123.4567)`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

}
