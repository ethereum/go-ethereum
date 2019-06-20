package goja

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

var hex = "0123456789abcdef"

func (r *Runtime) builtinJSON_parse(call FunctionCall) Value {
	d := json.NewDecoder(bytes.NewBufferString(call.Argument(0).String()))

	value, err := r.builtinJSON_decodeValue(d)
	if err != nil {
		panic(r.newError(r.global.SyntaxError, err.Error()))
	}

	if tok, err := d.Token(); err != io.EOF {
		panic(r.newError(r.global.SyntaxError, "Unexpected token at the end: %v", tok))
	}

	var reviver func(FunctionCall) Value

	if arg1 := call.Argument(1); arg1 != _undefined {
		reviver, _ = arg1.ToObject(r).self.assertCallable()
	}

	if reviver != nil {
		root := r.NewObject()
		root.self.putStr("", value, false)
		return r.builtinJSON_reviveWalk(reviver, root, stringEmpty)
	}

	return value
}

func (r *Runtime) builtinJSON_decodeToken(d *json.Decoder, tok json.Token) (Value, error) {
	switch tok := tok.(type) {
	case json.Delim:
		switch tok {
		case '{':
			return r.builtinJSON_decodeObject(d)
		case '[':
			return r.builtinJSON_decodeArray(d)
		}
	case nil:
		return _null, nil
	case string:
		return newStringValue(tok), nil
	case float64:
		return floatToValue(tok), nil
	case bool:
		if tok {
			return valueTrue, nil
		}
		return valueFalse, nil
	}
	return nil, fmt.Errorf("Unexpected token (%T): %v", tok, tok)
}

func (r *Runtime) builtinJSON_decodeValue(d *json.Decoder) (Value, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, err
	}
	return r.builtinJSON_decodeToken(d, tok)
}

func (r *Runtime) builtinJSON_decodeObject(d *json.Decoder) (*Object, error) {
	object := r.NewObject()
	for {
		key, end, err := r.builtinJSON_decodeObjectKey(d)
		if err != nil {
			return nil, err
		}
		if end {
			break
		}
		value, err := r.builtinJSON_decodeValue(d)
		if err != nil {
			return nil, err
		}

		if key == "__proto__" {
			descr := propertyDescr{
				Value:        value,
				Writable:     FLAG_TRUE,
				Enumerable:   FLAG_TRUE,
				Configurable: FLAG_TRUE,
			}
			object.self.defineOwnProperty(string__proto__, descr, false)
		} else {
			object.self.putStr(key, value, false)
		}
	}
	return object, nil
}

func (r *Runtime) builtinJSON_decodeObjectKey(d *json.Decoder) (string, bool, error) {
	tok, err := d.Token()
	if err != nil {
		return "", false, err
	}
	switch tok := tok.(type) {
	case json.Delim:
		if tok == '}' {
			return "", true, nil
		}
	case string:
		return tok, false, nil
	}

	return "", false, fmt.Errorf("Unexpected token (%T): %v", tok, tok)
}

func (r *Runtime) builtinJSON_decodeArray(d *json.Decoder) (*Object, error) {
	var arrayValue []Value
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := tok.(json.Delim); ok {
			if delim == ']' {
				break
			}
		}
		value, err := r.builtinJSON_decodeToken(d, tok)
		if err != nil {
			return nil, err
		}
		arrayValue = append(arrayValue, value)
	}
	return r.newArrayValues(arrayValue), nil
}

func isArray(object *Object) bool {
	switch object.self.className() {
	case classArray:
		return true
	default:
		return false
	}
}

func (r *Runtime) builtinJSON_reviveWalk(reviver func(FunctionCall) Value, holder *Object, name Value) Value {
	value := holder.self.get(name)
	if value == nil {
		value = _undefined
	}

	if object := value.(*Object); object != nil {
		if isArray(object) {
			length := object.self.getStr("length").ToInteger()
			for index := int64(0); index < length; index++ {
				name := intToValue(index)
				value := r.builtinJSON_reviveWalk(reviver, object, name)
				if value == _undefined {
					object.self.delete(name, false)
				} else {
					object.self.put(name, value, false)
				}
			}
		} else {
			for item, f := object.self.enumerate(false, false)(); f != nil; item, f = f() {
				value := r.builtinJSON_reviveWalk(reviver, object, name)
				if value == _undefined {
					object.self.deleteStr(item.name, false)
				} else {
					object.self.putStr(item.name, value, false)
				}
			}
		}
	}
	return reviver(FunctionCall{
		This:      holder,
		Arguments: []Value{name, value},
	})
}

type _builtinJSON_stringifyContext struct {
	r                *Runtime
	stack            []*Object
	propertyList     []Value
	replacerFunction func(FunctionCall) Value
	gap, indent      string
	buf              bytes.Buffer
}

func (r *Runtime) builtinJSON_stringify(call FunctionCall) Value {
	ctx := _builtinJSON_stringifyContext{
		r: r,
	}

	replacer, _ := call.Argument(1).(*Object)
	if replacer != nil {
		if isArray(replacer) {
			length := replacer.self.getStr("length").ToInteger()
			seen := map[string]bool{}
			propertyList := make([]Value, length)
			length = 0
			for index := range propertyList {
				var name string
				value := replacer.self.get(intToValue(int64(index)))
				if s, ok := value.assertString(); ok {
					name = s.String()
				} else if _, ok := value.assertInt(); ok {
					name = value.String()
				} else if _, ok := value.assertFloat(); ok {
					name = value.String()
				} else if o, ok := value.(*Object); ok {
					switch o.self.className() {
					case classNumber, classString:
						name = value.String()
					}
				}
				if seen[name] {
					continue
				}
				seen[name] = true
				length += 1
				propertyList[index] = newStringValue(name)
			}
			ctx.propertyList = propertyList[0:length]
		} else if c, ok := replacer.self.assertCallable(); ok {
			ctx.replacerFunction = c
		}
	}
	if spaceValue := call.Argument(2); spaceValue != _undefined {
		if o, ok := spaceValue.(*Object); ok {
			switch o := o.self.(type) {
			case *primitiveValueObject:
				spaceValue = o.pValue
			case *stringObject:
				spaceValue = o.value
			}
		}
		isNum := false
		var num int64
		num, isNum = spaceValue.assertInt()
		if !isNum {
			if f, ok := spaceValue.assertFloat(); ok {
				num = int64(f)
				isNum = true
			}
		}
		if isNum {
			if num > 0 {
				if num > 10 {
					num = 10
				}
				ctx.gap = strings.Repeat(" ", int(num))
			}
		} else {
			if s, ok := spaceValue.assertString(); ok {
				str := s.String()
				if len(str) > 10 {
					ctx.gap = str[:10]
				} else {
					ctx.gap = str
				}
			}
		}
	}

	if ctx.do(call.Argument(0)) {
		return newStringValue(ctx.buf.String())
	}
	return _undefined
}

func (ctx *_builtinJSON_stringifyContext) do(v Value) bool {
	holder := ctx.r.NewObject()
	holder.self.putStr("", v, false)
	return ctx.str(stringEmpty, holder)
}

func (ctx *_builtinJSON_stringifyContext) str(key Value, holder *Object) bool {
	value := holder.self.get(key)
	if value == nil {
		value = _undefined
	}

	if object, ok := value.(*Object); ok {
		if toJSON, ok := object.self.getStr("toJSON").(*Object); ok {
			if c, ok := toJSON.self.assertCallable(); ok {
				value = c(FunctionCall{
					This:      value,
					Arguments: []Value{key},
				})
			}
		}
	}

	if ctx.replacerFunction != nil {
		value = ctx.replacerFunction(FunctionCall{
			This:      holder,
			Arguments: []Value{key, value},
		})
	}

	if o, ok := value.(*Object); ok {
		switch o1 := o.self.(type) {
		case *primitiveValueObject:
			value = o1.pValue
		case *stringObject:
			value = o1.value
		case *objectGoReflect:
			if o1.toJson != nil {
				value = ctx.r.ToValue(o1.toJson())
			} else if v, ok := o1.origValue.Interface().(json.Marshaler); ok {
				b, err := v.MarshalJSON()
				if err != nil {
					panic(err)
				}
				ctx.buf.Write(b)
				return true
			} else {
				switch o1.className() {
				case classNumber:
					value = o1.toPrimitiveNumber()
				case classString:
					value = o1.toPrimitiveString()
				case classBoolean:
					if o.ToInteger() != 0 {
						value = valueTrue
					} else {
						value = valueFalse
					}
				}
			}
		}
	}

	switch value1 := value.(type) {
	case valueBool:
		if value1 {
			ctx.buf.WriteString("true")
		} else {
			ctx.buf.WriteString("false")
		}
	case valueString:
		ctx.quote(value1)
	case valueInt:
		ctx.buf.WriteString(value.String())
	case valueFloat:
		if !math.IsNaN(float64(value1)) && !math.IsInf(float64(value1), 0) {
			ctx.buf.WriteString(value.String())
		} else {
			ctx.buf.WriteString("null")
		}
	case valueNull:
		ctx.buf.WriteString("null")
	case *Object:
		for _, object := range ctx.stack {
			if value1 == object {
				ctx.r.typeErrorResult(true, "Converting circular structure to JSON")
			}
		}
		ctx.stack = append(ctx.stack, value1)
		defer func() { ctx.stack = ctx.stack[:len(ctx.stack)-1] }()
		if _, ok := value1.self.assertCallable(); !ok {
			if isArray(value1) {
				ctx.ja(value1)
			} else {
				ctx.jo(value1)
			}
		} else {
			return false
		}
	default:
		return false
	}
	return true
}

func (ctx *_builtinJSON_stringifyContext) ja(array *Object) {
	var stepback string
	if ctx.gap != "" {
		stepback = ctx.indent
		ctx.indent += ctx.gap
	}
	length := array.self.getStr("length").ToInteger()
	if length == 0 {
		ctx.buf.WriteString("[]")
		return
	}

	ctx.buf.WriteByte('[')
	var separator string
	if ctx.gap != "" {
		ctx.buf.WriteByte('\n')
		ctx.buf.WriteString(ctx.indent)
		separator = ",\n" + ctx.indent
	} else {
		separator = ","
	}

	for i := int64(0); i < length; i++ {
		if !ctx.str(intToValue(i), array) {
			ctx.buf.WriteString("null")
		}
		if i < length-1 {
			ctx.buf.WriteString(separator)
		}
	}
	if ctx.gap != "" {
		ctx.buf.WriteByte('\n')
		ctx.buf.WriteString(stepback)
		ctx.indent = stepback
	}
	ctx.buf.WriteByte(']')
}

func (ctx *_builtinJSON_stringifyContext) jo(object *Object) {
	var stepback string
	if ctx.gap != "" {
		stepback = ctx.indent
		ctx.indent += ctx.gap
	}

	ctx.buf.WriteByte('{')
	mark := ctx.buf.Len()
	var separator string
	if ctx.gap != "" {
		ctx.buf.WriteByte('\n')
		ctx.buf.WriteString(ctx.indent)
		separator = ",\n" + ctx.indent
	} else {
		separator = ","
	}

	var props []Value
	if ctx.propertyList == nil {
		for item, f := object.self.enumerate(false, false)(); f != nil; item, f = f() {
			props = append(props, newStringValue(item.name))
		}
	} else {
		props = ctx.propertyList
	}

	empty := true
	for _, name := range props {
		off := ctx.buf.Len()
		if !empty {
			ctx.buf.WriteString(separator)
		}
		ctx.quote(name.ToString())
		if ctx.gap != "" {
			ctx.buf.WriteString(": ")
		} else {
			ctx.buf.WriteByte(':')
		}
		if ctx.str(name, object) {
			if empty {
				empty = false
			}
		} else {
			ctx.buf.Truncate(off)
		}
	}

	if empty {
		ctx.buf.Truncate(mark)
	} else {
		if ctx.gap != "" {
			ctx.buf.WriteByte('\n')
			ctx.buf.WriteString(stepback)
			ctx.indent = stepback
		}
	}
	ctx.buf.WriteByte('}')
}

func (ctx *_builtinJSON_stringifyContext) quote(str valueString) {
	ctx.buf.WriteByte('"')
	reader := str.reader(0)
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			break
		}
		switch r {
		case '"', '\\':
			ctx.buf.WriteByte('\\')
			ctx.buf.WriteByte(byte(r))
		case 0x08:
			ctx.buf.WriteString(`\b`)
		case 0x09:
			ctx.buf.WriteString(`\t`)
		case 0x0A:
			ctx.buf.WriteString(`\n`)
		case 0x0C:
			ctx.buf.WriteString(`\f`)
		case 0x0D:
			ctx.buf.WriteString(`\r`)
		default:
			if r < 0x20 {
				ctx.buf.WriteString(`\u00`)
				ctx.buf.WriteByte(hex[r>>4])
				ctx.buf.WriteByte(hex[r&0xF])
			} else {
				ctx.buf.WriteRune(r)
			}
		}
	}
	ctx.buf.WriteByte('"')
}

func (r *Runtime) initJSON() {
	JSON := r.newBaseObject(r.global.ObjectPrototype, "JSON")
	JSON._putProp("parse", r.newNativeFunc(r.builtinJSON_parse, nil, "parse", nil, 2), true, false, true)
	JSON._putProp("stringify", r.newNativeFunc(r.builtinJSON_stringify, nil, "stringify", nil, 3), true, false, true)

	r.addToGlobal("JSON", JSON.val)
}
