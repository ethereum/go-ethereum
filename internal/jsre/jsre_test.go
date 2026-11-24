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

package jsre

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dop251/goja"
)

type testNativeObjectBinding struct {
	vm *goja.Runtime
}

type msg struct {
	Msg string
}

func (no *testNativeObjectBinding) TestMethod(call goja.FunctionCall) goja.Value {
	m := call.Argument(0).ToString().String()
	return no.vm.ToValue(&msg{m})
}

func newWithTestJS(t *testing.T, testjs string) *JSRE {
	dir := t.TempDir()
	if testjs != "" {
		if err := os.WriteFile(filepath.Join(dir, "test.js"), []byte(testjs), os.ModePerm); err != nil {
			t.Fatal("cannot create test.js:", err)
		}
	}
	jsre := New(dir, os.Stdout)
	return jsre
}

func TestExec(t *testing.T) {
	t.Parallel()

	jsre := newWithTestJS(t, `msg = "testMsg"`)

	err := jsre.Exec("test.js")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	val, err := jsre.Run("msg")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if val.ExportType().Kind() != reflect.String {
		t.Errorf("expected string value, got %v", val)
	}
	exp := "testMsg"
	got := val.ToString().String()
	if exp != got {
		t.Errorf("expected '%v', got '%v'", exp, got)
	}
	jsre.Stop(false)
}

func TestNatto(t *testing.T) {
	t.Parallel()

	jsre := newWithTestJS(t, `setTimeout(function(){msg = "testMsg"}, 1);`)

	err := jsre.Exec("test.js")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	val, err := jsre.Run("msg")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val.ExportType().Kind() != reflect.String {
		t.Fatalf("expected string value, got %v", val)
	}
	exp := "testMsg"
	got := val.ToString().String()
	if exp != got {
		t.Fatalf("expected '%v', got '%v'", exp, got)
	}
	jsre.Stop(false)
}

func TestBind(t *testing.T) {
	t.Parallel()

	jsre := New("", os.Stdout)
	defer jsre.Stop(false)

	jsre.Set("no", &testNativeObjectBinding{vm: jsre.vm})

	_, err := jsre.Run(`no.TestMethod("testMsg")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLoadScript(t *testing.T) {
	t.Parallel()

	jsre := newWithTestJS(t, `msg = "testMsg"`)

	_, err := jsre.Run(`loadScript("test.js")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	val, err := jsre.Run("msg")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if val.ExportType().Kind() != reflect.String {
		t.Errorf("expected string value, got %v", val)
	}
	exp := "testMsg"
	got := val.ToString().String()
	if exp != got {
		t.Errorf("expected '%v', got '%v'", exp, got)
	}
	jsre.Stop(false)
}

// waitFor polls the JS expression cond until it evaluates to true.
func waitFor(t *testing.T, re *JSRE, cond string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		val, err := re.Run(cond)
		if err != nil {
			t.Fatalf("evaluating %q: %v", cond, err)
		}
		if val.ToBoolean() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q", cond)
}

func TestTimerArguments(t *testing.T) {
	t.Parallel()

	re := New("", os.Stdout)
	defer re.Stop(false)

	// Timers without a delay.
	if _, err := re.Run(`
		var timeoutRan = false;
		setTimeout(function() { timeoutRan = true });

		var intervalRuns = 0;
		var iv = setInterval(function() {
			if (++intervalRuns == 2) {
				clearInterval(iv);
			}
		});
	`); err != nil {
		t.Fatal(err)
	}
	waitFor(t, re, `timeoutRan && intervalRuns == 2`)

	// Extra arguments are passed to the callback, and only those.
	if _, err := re.Run(`
		var extraArgs, noArgs;
		setTimeout(function() { extraArgs = Array.prototype.slice.call(arguments) }, 1, 2, "x");
		setTimeout(function() { noArgs = arguments.length }, 1);
	`); err != nil {
		t.Fatal(err)
	}
	waitFor(t, re, `extraArgs !== undefined && noArgs !== undefined`)
	if val, err := re.Run(`JSON.stringify(extraArgs) + "/" + noArgs`); err != nil {
		t.Fatal(err)
	} else if got, want := val.String(), `[2,"x"]/0`; got != want {
		t.Fatalf("wrong callback arguments: got %s, want %s", got, want)
	}

	// A callback that is not a function throws a catchable TypeError.
	for _, expr := range []string{
		`setTimeout("x", 1)`,
		`setTimeout(null)`,
		`setInterval({}, 1)`,
		`setInterval(42)`,
	} {
		val, err := re.Run(`(function() {
			try { ` + expr + `; } catch (e) { return e instanceof TypeError; }
			return false;
		})()`)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if !val.ToBoolean() {
			t.Fatalf("%s: expected TypeError", expr)
		}
	}

	// The event loop is still running.
	time.Sleep(20 * time.Millisecond)
	if val, err := re.Run(`1 + 1`); err != nil || val.ToInteger() != 2 {
		t.Fatalf("JSRE not usable after timers: %v, %v", val, err)
	}
}
