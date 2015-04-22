package jsre

import (
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"os"
	"testing"
	"time"
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

	ioutil.WriteFile("/tmp/test.js", []byte(`msg = "testMsg"`), os.ModePerm)
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
	jsre.Stop(false)
}

func TestNatto(t *testing.T) {
	jsre := New("/tmp")

	ioutil.WriteFile("/tmp/test.js", []byte(`setTimeout(function(){msg = "testMsg"}, 1);`), os.ModePerm)
	err := jsre.Exec("test.js")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	time.Sleep(time.Millisecond * 10)
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
	jsre.Stop(false)
}

func TestBind(t *testing.T) {
	jsre := New("/tmp")

	jsre.Bind("no", &testNativeObjectBinding{jsre.ToVal})

	val, err := jsre.Run(`no.TestMethod("testMsg")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	pp, err := jsre.PrettyPrint(val)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("no: %v", pp)
	jsre.Stop(false)
}

func TestLoadScript(t *testing.T) {
	jsre := New("/tmp")

	ioutil.WriteFile("/tmp/test.js", []byte(`msg = "testMsg"`), os.ModePerm)
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
	jsre.Stop(false)
}
