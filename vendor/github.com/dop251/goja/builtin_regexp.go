package goja

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/dop251/goja/parser"
	"regexp"
)

func (r *Runtime) newRegexpObject(proto *Object) *regexpObject {
	v := &Object{runtime: r}

	o := &regexpObject{}
	o.class = classRegExp
	o.val = v
	o.extensible = true
	v.self = o
	o.prototype = proto
	o.init()
	return o
}

func (r *Runtime) newRegExpp(pattern regexpPattern, patternStr valueString, global, ignoreCase, multiline bool, proto *Object) *Object {
	o := r.newRegexpObject(proto)

	o.pattern = pattern
	o.source = patternStr
	o.global = global
	o.ignoreCase = ignoreCase
	o.multiline = multiline

	return o.val
}

func compileRegexp(patternStr, flags string) (p regexpPattern, global, ignoreCase, multiline bool, err error) {

	if flags != "" {
		invalidFlags := func() {
			err = fmt.Errorf("Invalid flags supplied to RegExp constructor '%s'", flags)
		}
		for _, chr := range flags {
			switch chr {
			case 'g':
				if global {
					invalidFlags()
					return
				}
				global = true
			case 'm':
				if multiline {
					invalidFlags()
					return
				}
				multiline = true
			case 'i':
				if ignoreCase {
					invalidFlags()
					return
				}
				ignoreCase = true
			default:
				invalidFlags()
				return
			}
		}
	}

	re2Str, err1 := parser.TransformRegExp(patternStr)
	if /*false &&*/ err1 == nil {
		re2flags := ""
		if multiline {
			re2flags += "m"
		}
		if ignoreCase {
			re2flags += "i"
		}
		if len(re2flags) > 0 {
			re2Str = fmt.Sprintf("(?%s:%s)", re2flags, re2Str)
		}

		pattern, err1 := regexp.Compile(re2Str)
		if err1 != nil {
			err = fmt.Errorf("Invalid regular expression (re2): %s (%v)", re2Str, err1)
			return
		}

		p = (*regexpWrapper)(pattern)
	} else {
		var opts regexp2.RegexOptions = regexp2.ECMAScript
		if multiline {
			opts |= regexp2.Multiline
		}
		if ignoreCase {
			opts |= regexp2.IgnoreCase
		}
		regexp2Pattern, err1 := regexp2.Compile(patternStr, opts)
		if err1 != nil {
			err = fmt.Errorf("Invalid regular expression (regexp2): %s (%v)", patternStr, err1)
			return
		}
		p = (*regexp2Wrapper)(regexp2Pattern)
	}
	return
}

func (r *Runtime) newRegExp(patternStr valueString, flags string, proto *Object) *Object {
	pattern, global, ignoreCase, multiline, err := compileRegexp(patternStr.String(), flags)
	if err != nil {
		panic(r.newSyntaxError(err.Error(), -1))
	}
	return r.newRegExpp(pattern, patternStr, global, ignoreCase, multiline, proto)
}

func (r *Runtime) builtin_newRegExp(args []Value) *Object {
	var pattern valueString
	var flags string
	if len(args) > 0 {
		if obj, ok := args[0].(*Object); ok {
			if regexp, ok := obj.self.(*regexpObject); ok {
				if len(args) < 2 || args[1] == _undefined {
					return regexp.clone()
				} else {
					return r.newRegExp(regexp.source, args[1].String(), r.global.RegExpPrototype)
				}
			}
		}
		if args[0] != _undefined {
			pattern = args[0].ToString()
		}
	}
	if len(args) > 1 {
		if a := args[1]; a != _undefined {
			flags = a.String()
		}
	}
	if pattern == nil {
		pattern = stringEmpty
	}
	return r.newRegExp(pattern, flags, r.global.RegExpPrototype)
}

func (r *Runtime) builtin_RegExp(call FunctionCall) Value {
	flags := call.Argument(1)
	if flags == _undefined {
		if obj, ok := call.Argument(0).(*Object); ok {
			if _, ok := obj.self.(*regexpObject); ok {
				return call.Arguments[0]
			}
		}
	}
	return r.builtin_newRegExp(call.Arguments)
}

func (r *Runtime) regexpproto_exec(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		return this.exec(call.Argument(0).ToString())
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.exec called on incompatible receiver %s", call.This.ToString())
		return nil
	}
}

func (r *Runtime) regexpproto_test(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.test(call.Argument(0).ToString()) {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.test called on incompatible receiver %s", call.This.ToString())
		return nil
	}
}

func (r *Runtime) regexpproto_toString(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		var g, i, m string
		if this.global {
			g = "g"
		}
		if this.ignoreCase {
			i = "i"
		}
		if this.multiline {
			m = "m"
		}
		return newStringValue(fmt.Sprintf("/%s/%s%s%s", this.source.String(), g, i, m))
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.toString called on incompatible receiver %s", call.This)
		return nil
	}
}

func (r *Runtime) regexpproto_getSource(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		return this.source
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.source getter called on incompatible receiver %s", call.This.ToString())
		return nil
	}
}

func (r *Runtime) regexpproto_getGlobal(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.global {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.global getter called on incompatible receiver %s", call.This.ToString())
		return nil
	}
}

func (r *Runtime) regexpproto_getMultiline(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.multiline {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.multiline getter called on incompatible receiver %s", call.This.ToString())
		return nil
	}
}

func (r *Runtime) regexpproto_getIgnoreCase(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.ignoreCase {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.ignoreCase getter called on incompatible receiver %s", call.This.ToString())
		return nil
	}
}

func (r *Runtime) initRegExp() {
	r.global.RegExpPrototype = r.NewObject()
	o := r.global.RegExpPrototype.self
	o._putProp("exec", r.newNativeFunc(r.regexpproto_exec, nil, "exec", nil, 1), true, false, true)
	o._putProp("test", r.newNativeFunc(r.regexpproto_test, nil, "test", nil, 1), true, false, true)
	o._putProp("toString", r.newNativeFunc(r.regexpproto_toString, nil, "toString", nil, 0), true, false, true)
	o.putStr("source", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getSource, nil, "get source", nil, 0),
		accessor:     true,
	}, false)
	o.putStr("global", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getGlobal, nil, "get global", nil, 0),
		accessor:     true,
	}, false)
	o.putStr("multiline", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getMultiline, nil, "get multiline", nil, 0),
		accessor:     true,
	}, false)
	o.putStr("ignoreCase", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getIgnoreCase, nil, "get ignoreCase", nil, 0),
		accessor:     true,
	}, false)

	r.global.RegExp = r.newNativeFunc(r.builtin_RegExp, r.builtin_newRegExp, "RegExp", r.global.RegExpPrototype, 2)
	r.addToGlobal("RegExp", r.global.RegExp)
}
