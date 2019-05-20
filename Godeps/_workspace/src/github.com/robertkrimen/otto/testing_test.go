package otto

import (
	"./terst"
	"errors"
	"strings"
	"testing"
	"time"
)

func tt(t *testing.T, arguments ...func()) {
	halt := errors.New("A test was taking too long")
	timer := time.AfterFunc(2*time.Second, func() {
		panic(halt)
	})
	defer func() {
		timer.Stop()
	}()
	terst.Terst(t, arguments...)
}

func is(arguments ...interface{}) bool {
	var got, expect interface{}

	switch len(arguments) {
	case 0, 1:
		return terst.Is(arguments...)
	case 2:
		got, expect = arguments[0], arguments[1]
	default:
		got, expect = arguments[0], arguments[2]
	}

	switch value := got.(type) {
	case Value:
		if value.value != nil {
			got = value.value
		}
	case *Error:
		if value != nil {
			got = value.Error()
		}
		if expect == nil {
			// FIXME This is weird
			expect = ""
		}
	}

	if len(arguments) == 2 {
		arguments[0] = got
		arguments[1] = expect
	} else {
		arguments[0] = got
		arguments[2] = expect
	}

	return terst.Is(arguments...)
}

func test(arguments ...interface{}) (func(string, ...interface{}) Value, *_tester) {
	tester := newTester()
	if len(arguments) > 0 {
		tester.test(arguments[0].(string))
	}
	return tester.test, tester
}

type _tester struct {
	vm *Otto
}

func newTester() *_tester {
	return &_tester{
		vm: New(),
	}
}

func (self *_tester) Get(name string) (Value, error) {
	return self.vm.Get(name)
}

func (self *_tester) Set(name string, value interface{}) Value {
	err := self.vm.Set(name, value)
	is(err, nil)
	if err != nil {
		terst.Caller().T().FailNow()
	}
	return self.vm.getValue(name)
}

func (self *_tester) Run(src interface{}) (Value, error) {
	return self.vm.Run(src)
}

func (self *_tester) test(name string, expect ...interface{}) Value {
	vm := self.vm
	raise := false
	defer func() {
		if caught := recover(); caught != nil {
			if exception, ok := caught.(*_exception); ok {
				caught = exception.eject()
			}
			if raise {
				if len(expect) > 0 {
					is(caught, expect[0])
				}
			} else {
				dbg("Panic, caught:", caught)
				panic(caught)
			}
		}
	}()
	var value Value
	var err error
	if isIdentifier(name) {
		value = vm.getValue(name)
	} else {
		source := name
		index := strings.Index(source, "raise:")
		if index == 0 {
			raise = true
			source = source[6:]
			source = strings.TrimLeft(source, " ")
		}
		value, err = vm.runtime.cmpl_run(source)
		if err != nil {
			panic(err)
		}
	}
	value = value.resolve()
	if len(expect) > 0 {
		is(value, expect[0])
	}
	return value
}
