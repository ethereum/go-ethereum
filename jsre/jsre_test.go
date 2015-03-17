package jsre

import (
	"github.com/obscuren/otto"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

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

	common.WriteFile("/tmp/test.js", []byte(`msg = "testMsg"`))
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
	exp := "testMsg"
	got, _ := val.ToString()
	if exp != got {
		t.Errorf("expected '%v', got '%v'", exp, got)
	}
}

func TestBind(t *testing.T) {
	jsre := New("/tmp")

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

func TestLoadScript(t *testing.T) {
	jsre := New("/tmp")

	common.WriteFile("/tmp/test.js", []byte(`msg = "testMsg"`))
	_, err := jsre.Run(`loadScript("test.js")`)
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
	exp := "testMsg"
	got, _ := val.ToString()
	if exp != got {
		t.Errorf("expected '%v', got '%v'", exp, got)
	}
}
