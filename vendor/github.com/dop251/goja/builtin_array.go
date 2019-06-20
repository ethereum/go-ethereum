package goja

import (
	"bytes"
	"sort"
	"strings"
)

func (r *Runtime) builtin_newArray(args []Value, proto *Object) *Object {
	l := len(args)
	if l == 1 {
		if al, ok := args[0].assertInt(); ok {
			return r.newArrayLength(al)
		} else if f, ok := args[0].assertFloat(); ok {
			al := int64(f)
			if float64(al) == f {
				return r.newArrayLength(al)
			} else {
				panic(r.newError(r.global.RangeError, "Invalid array length"))
			}
		}
		return r.newArrayValues([]Value{args[0]})
	} else {
		argsCopy := make([]Value, l)
		copy(argsCopy, args)
		return r.newArrayValues(argsCopy)
	}
}

func (r *Runtime) generic_push(obj *Object, call FunctionCall) Value {
	l := toLength(obj.self.getStr("length"))
	nl := l + int64(len(call.Arguments))
	if nl >= maxInt {
		r.typeErrorResult(true, "Invalid array length")
		panic("unreachable")
	}
	for i, arg := range call.Arguments {
		obj.self.put(intToValue(l+int64(i)), arg, true)
	}
	n := intToValue(nl)
	obj.self.putStr("length", n, true)
	return n
}

func (r *Runtime) arrayproto_push(call FunctionCall) Value {
	obj := call.This.ToObject(r)
	return r.generic_push(obj, call)
}

func (r *Runtime) arrayproto_pop_generic(obj *Object, call FunctionCall) Value {
	l := toLength(obj.self.getStr("length"))
	if l == 0 {
		obj.self.putStr("length", intToValue(0), true)
		return _undefined
	}
	idx := intToValue(l - 1)
	val := obj.self.get(idx)
	obj.self.delete(idx, true)
	obj.self.putStr("length", idx, true)
	return val
}

func (r *Runtime) arrayproto_pop(call FunctionCall) Value {
	obj := call.This.ToObject(r)
	if a, ok := obj.self.(*arrayObject); ok {
		l := a.length
		if l > 0 {
			var val Value
			l--
			if l < int64(len(a.values)) {
				val = a.values[l]
			}
			if val == nil {
				// optimisation bail-out
				return r.arrayproto_pop_generic(obj, call)
			}
			if _, ok := val.(*valueProperty); ok {
				// optimisation bail-out
				return r.arrayproto_pop_generic(obj, call)
			}
			//a._setLengthInt(l, false)
			a.values[l] = nil
			a.values = a.values[:l]
			a.length = l
			return val
		}
		return _undefined
	} else {
		return r.arrayproto_pop_generic(obj, call)
	}
}

func (r *Runtime) arrayproto_join(call FunctionCall) Value {
	o := call.This.ToObject(r)
	l := int(toLength(o.self.getStr("length")))
	sep := ""
	if s := call.Argument(0); s != _undefined {
		sep = s.String()
	} else {
		sep = ","
	}
	if l == 0 {
		return stringEmpty
	}

	var buf bytes.Buffer

	element0 := o.self.get(intToValue(0))
	if element0 != nil && element0 != _undefined && element0 != _null {
		buf.WriteString(element0.String())
	}

	for i := 1; i < l; i++ {
		buf.WriteString(sep)
		element := o.self.get(intToValue(int64(i)))
		if element != nil && element != _undefined && element != _null {
			buf.WriteString(element.String())
		}
	}

	return newStringValue(buf.String())
}

func (r *Runtime) arrayproto_toString(call FunctionCall) Value {
	array := call.This.ToObject(r)
	f := array.self.getStr("join")
	if fObj, ok := f.(*Object); ok {
		if fcall, ok := fObj.self.assertCallable(); ok {
			return fcall(FunctionCall{
				This: array,
			})
		}
	}
	return r.objectproto_toString(FunctionCall{
		This: array,
	})
}

func (r *Runtime) writeItemLocaleString(item Value, buf *bytes.Buffer) {
	if item != nil && item != _undefined && item != _null {
		itemObj := item.ToObject(r)
		if f, ok := itemObj.self.getStr("toLocaleString").(*Object); ok {
			if c, ok := f.self.assertCallable(); ok {
				strVal := c(FunctionCall{
					This: itemObj,
				})
				buf.WriteString(strVal.String())
				return
			}
		}
		r.typeErrorResult(true, "Property 'toLocaleString' of object %s is not a function", itemObj)
	}
}

func (r *Runtime) arrayproto_toLocaleString_generic(obj *Object, start int64, buf *bytes.Buffer) Value {
	length := toLength(obj.self.getStr("length"))
	for i := int64(start); i < length; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		item := obj.self.get(intToValue(i))
		r.writeItemLocaleString(item, buf)
	}
	return newStringValue(buf.String())
}

func (r *Runtime) arrayproto_toLocaleString(call FunctionCall) Value {
	array := call.This.ToObject(r)
	if a, ok := array.self.(*arrayObject); ok {
		var buf bytes.Buffer
		for i := int64(0); i < a.length; i++ {
			var item Value
			if i < int64(len(a.values)) {
				item = a.values[i]
			}
			if item == nil {
				return r.arrayproto_toLocaleString_generic(array, i, &buf)
			}
			if prop, ok := item.(*valueProperty); ok {
				item = prop.get(array)
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			r.writeItemLocaleString(item, &buf)
		}
		return newStringValue(buf.String())
	} else {
		return r.arrayproto_toLocaleString_generic(array, 0, bytes.NewBuffer(nil))
	}

}

func (r *Runtime) arrayproto_concat_append(a *Object, item Value) {
	descr := propertyDescr{
		Writable:     FLAG_TRUE,
		Enumerable:   FLAG_TRUE,
		Configurable: FLAG_TRUE,
	}

	aLength := toLength(a.self.getStr("length"))
	if obj, ok := item.(*Object); ok {
		if isArray(obj) {
			length := toLength(obj.self.getStr("length"))
			for i := int64(0); i < length; i++ {
				v := obj.self.get(intToValue(i))
				if v != nil {
					descr.Value = v
					a.self.defineOwnProperty(intToValue(aLength), descr, false)
					aLength++
				} else {
					aLength++
					a.self.putStr("length", intToValue(aLength), false)
				}
			}
			return
		}
	}
	descr.Value = item
	a.self.defineOwnProperty(intToValue(aLength), descr, false)
}

func (r *Runtime) arrayproto_concat(call FunctionCall) Value {
	a := r.newArrayValues(nil)
	r.arrayproto_concat_append(a, call.This.ToObject(r))
	for _, item := range call.Arguments {
		r.arrayproto_concat_append(a, item)
	}
	return a
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (r *Runtime) arrayproto_slice(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	start := call.Argument(0).ToInteger()
	if start < 0 {
		start = max(length+start, 0)
	} else {
		start = min(start, length)
	}
	var end int64
	if endArg := call.Argument(1); endArg != _undefined {
		end = endArg.ToInteger()
	} else {
		end = length
	}
	if end < 0 {
		end = max(length+end, 0)
	} else {
		end = min(end, length)
	}

	count := end - start
	if count < 0 {
		count = 0
	}
	a := r.newArrayLength(count)

	n := int64(0)
	descr := propertyDescr{
		Writable:     FLAG_TRUE,
		Enumerable:   FLAG_TRUE,
		Configurable: FLAG_TRUE,
	}
	for start < end {
		p := o.self.get(intToValue(start))
		if p != nil && p != _undefined {
			descr.Value = p
			a.self.defineOwnProperty(intToValue(n), descr, false)
		}
		start++
		n++
	}
	return a
}

func (r *Runtime) arrayproto_sort(call FunctionCall) Value {
	o := call.This.ToObject(r)

	var compareFn func(FunctionCall) Value

	if arg, ok := call.Argument(0).(*Object); ok {
		compareFn, _ = arg.self.assertCallable()
	}

	ctx := arraySortCtx{
		obj:     o.self,
		compare: compareFn,
	}

	sort.Sort(&ctx)
	return o
}

func (r *Runtime) arrayproto_splice(call FunctionCall) Value {
	o := call.This.ToObject(r)
	a := r.newArrayValues(nil)
	length := toLength(o.self.getStr("length"))
	relativeStart := call.Argument(0).ToInteger()
	var actualStart int64
	if relativeStart < 0 {
		actualStart = max(length+relativeStart, 0)
	} else {
		actualStart = min(relativeStart, length)
	}

	actualDeleteCount := min(max(call.Argument(1).ToInteger(), 0), length-actualStart)

	for k := int64(0); k < actualDeleteCount; k++ {
		from := intToValue(k + actualStart)
		if o.self.hasProperty(from) {
			a.self.put(intToValue(k), o.self.get(from), false)
		}
	}

	itemCount := max(int64(len(call.Arguments)-2), 0)
	if itemCount < actualDeleteCount {
		for k := actualStart; k < length-actualDeleteCount; k++ {
			from := intToValue(k + actualDeleteCount)
			to := intToValue(k + itemCount)
			if o.self.hasProperty(from) {
				o.self.put(to, o.self.get(from), true)
			} else {
				o.self.delete(to, true)
			}
		}

		for k := length; k > length-actualDeleteCount+itemCount; k-- {
			o.self.delete(intToValue(k-1), true)
		}
	} else if itemCount > actualDeleteCount {
		for k := length - actualDeleteCount; k > actualStart; k-- {
			from := intToValue(k + actualDeleteCount - 1)
			to := intToValue(k + itemCount - 1)
			if o.self.hasProperty(from) {
				o.self.put(to, o.self.get(from), true)
			} else {
				o.self.delete(to, true)
			}
		}
	}

	if itemCount > 0 {
		for i, item := range call.Arguments[2:] {
			o.self.put(intToValue(actualStart+int64(i)), item, true)
		}
	}

	o.self.putStr("length", intToValue(length-actualDeleteCount+itemCount), true)

	return a
}

func (r *Runtime) arrayproto_unshift(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	argCount := int64(len(call.Arguments))
	for k := length - 1; k >= 0; k-- {
		from := intToValue(k)
		to := intToValue(k + argCount)
		if o.self.hasProperty(from) {
			o.self.put(to, o.self.get(from), true)
		} else {
			o.self.delete(to, true)
		}
	}

	for k, arg := range call.Arguments {
		o.self.put(intToValue(int64(k)), arg, true)
	}

	newLen := intToValue(length + argCount)
	o.self.putStr("length", newLen, true)
	return newLen
}

func (r *Runtime) arrayproto_indexOf(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	if length == 0 {
		return intToValue(-1)
	}

	n := call.Argument(1).ToInteger()
	if n >= length {
		return intToValue(-1)
	}

	if n < 0 {
		n = max(length+n, 0)
	}

	searchElement := call.Argument(0)

	for ; n < length; n++ {
		idx := intToValue(n)
		if val := o.self.get(idx); val != nil {
			if searchElement.StrictEquals(val) {
				return idx
			}
		}
	}

	return intToValue(-1)
}

func (r *Runtime) arrayproto_lastIndexOf(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	if length == 0 {
		return intToValue(-1)
	}

	var fromIndex int64

	if len(call.Arguments) < 2 {
		fromIndex = length - 1
	} else {
		fromIndex = call.Argument(1).ToInteger()
		if fromIndex >= 0 {
			fromIndex = min(fromIndex, length-1)
		} else {
			fromIndex += length
		}
	}

	searchElement := call.Argument(0)

	for k := fromIndex; k >= 0; k-- {
		idx := intToValue(k)
		if val := o.self.get(idx); val != nil {
			if searchElement.StrictEquals(val) {
				return idx
			}
		}
	}

	return intToValue(-1)
}

func (r *Runtime) arrayproto_every(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		fc := FunctionCall{
			This:      call.Argument(1),
			Arguments: []Value{nil, nil, o},
		}
		for k := int64(0); k < length; k++ {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[0] = val
				fc.Arguments[1] = idx
				if !callbackFn(fc).ToBoolean() {
					return valueFalse
				}
			}
		}
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	return valueTrue
}

func (r *Runtime) arrayproto_some(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		fc := FunctionCall{
			This:      call.Argument(1),
			Arguments: []Value{nil, nil, o},
		}
		for k := int64(0); k < length; k++ {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[0] = val
				fc.Arguments[1] = idx
				if callbackFn(fc).ToBoolean() {
					return valueTrue
				}
			}
		}
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	return valueFalse
}

func (r *Runtime) arrayproto_forEach(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		fc := FunctionCall{
			This:      call.Argument(1),
			Arguments: []Value{nil, nil, o},
		}
		for k := int64(0); k < length; k++ {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[0] = val
				fc.Arguments[1] = idx
				callbackFn(fc)
			}
		}
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	return _undefined
}

func (r *Runtime) arrayproto_map(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		fc := FunctionCall{
			This:      call.Argument(1),
			Arguments: []Value{nil, nil, o},
		}
		a := r.newArrayObject()
		a._setLengthInt(length, true)
		a.values = make([]Value, length)
		for k := int64(0); k < length; k++ {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[0] = val
				fc.Arguments[1] = idx
				a.values[k] = callbackFn(fc)
				a.objCount++
			}
		}
		return a.val
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	panic("unreachable")
}

func (r *Runtime) arrayproto_filter(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		a := r.newArrayObject()
		fc := FunctionCall{
			This:      call.Argument(1),
			Arguments: []Value{nil, nil, o},
		}
		for k := int64(0); k < length; k++ {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[0] = val
				fc.Arguments[1] = idx
				if callbackFn(fc).ToBoolean() {
					a.values = append(a.values, val)
				}
			}
		}
		a.length = int64(len(a.values))
		a.objCount = a.length
		return a.val
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	panic("unreachable")
}

func (r *Runtime) arrayproto_reduce(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		fc := FunctionCall{
			This:      _undefined,
			Arguments: []Value{nil, nil, nil, o},
		}

		var k int64

		if len(call.Arguments) >= 2 {
			fc.Arguments[0] = call.Argument(1)
		} else {
			for ; k < length; k++ {
				idx := intToValue(k)
				if val := o.self.get(idx); val != nil {
					fc.Arguments[0] = val
					break
				}
			}
			if fc.Arguments[0] == nil {
				r.typeErrorResult(true, "No initial value")
				panic("unreachable")
			}
			k++
		}

		for ; k < length; k++ {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[1] = val
				fc.Arguments[2] = idx
				fc.Arguments[0] = callbackFn(fc)
			}
		}
		return fc.Arguments[0]
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	panic("unreachable")
}

func (r *Runtime) arrayproto_reduceRight(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	callbackFn := call.Argument(0).ToObject(r)
	if callbackFn, ok := callbackFn.self.assertCallable(); ok {
		fc := FunctionCall{
			This:      _undefined,
			Arguments: []Value{nil, nil, nil, o},
		}

		k := length - 1

		if len(call.Arguments) >= 2 {
			fc.Arguments[0] = call.Argument(1)
		} else {
			for ; k >= 0; k-- {
				idx := intToValue(k)
				if val := o.self.get(idx); val != nil {
					fc.Arguments[0] = val
					break
				}
			}
			if fc.Arguments[0] == nil {
				r.typeErrorResult(true, "No initial value")
				panic("unreachable")
			}
			k--
		}

		for ; k >= 0; k-- {
			idx := intToValue(k)
			if val := o.self.get(idx); val != nil {
				fc.Arguments[1] = val
				fc.Arguments[2] = idx
				fc.Arguments[0] = callbackFn(fc)
			}
		}
		return fc.Arguments[0]
	} else {
		r.typeErrorResult(true, "%s is not a function", call.Argument(0))
	}
	panic("unreachable")
}

func arrayproto_reverse_generic_step(o *Object, lower, upper int64) {
	lowerP := intToValue(lower)
	upperP := intToValue(upper)
	lowerValue := o.self.get(lowerP)
	upperValue := o.self.get(upperP)
	if lowerValue != nil && upperValue != nil {
		o.self.put(lowerP, upperValue, true)
		o.self.put(upperP, lowerValue, true)
	} else if lowerValue == nil && upperValue != nil {
		o.self.put(lowerP, upperValue, true)
		o.self.delete(upperP, true)
	} else if lowerValue != nil && upperValue == nil {
		o.self.delete(lowerP, true)
		o.self.put(upperP, lowerValue, true)
	}
}

func (r *Runtime) arrayproto_reverse_generic(o *Object, start int64) {
	l := toLength(o.self.getStr("length"))
	middle := l / 2
	for lower := start; lower != middle; lower++ {
		arrayproto_reverse_generic_step(o, lower, l-lower-1)
	}
}

func (r *Runtime) arrayproto_reverse(call FunctionCall) Value {
	o := call.This.ToObject(r)
	if a, ok := o.self.(*arrayObject); ok {
		l := a.length
		middle := l / 2
		al := int64(len(a.values))
		for lower := int64(0); lower != middle; lower++ {
			upper := l - lower - 1
			var lowerValue, upperValue Value
			if upper >= al || lower >= al {
				goto bailout
			}
			lowerValue = a.values[lower]
			if lowerValue == nil {
				goto bailout
			}
			if _, ok := lowerValue.(*valueProperty); ok {
				goto bailout
			}
			upperValue = a.values[upper]
			if upperValue == nil {
				goto bailout
			}
			if _, ok := upperValue.(*valueProperty); ok {
				goto bailout
			}

			a.values[lower], a.values[upper] = upperValue, lowerValue
			continue
		bailout:
			arrayproto_reverse_generic_step(o, lower, upper)
		}
		//TODO: go arrays
	} else {
		r.arrayproto_reverse_generic(o, 0)
	}
	return o
}

func (r *Runtime) arrayproto_shift(call FunctionCall) Value {
	o := call.This.ToObject(r)
	length := toLength(o.self.getStr("length"))
	if length == 0 {
		o.self.putStr("length", intToValue(0), true)
		return _undefined
	}
	first := o.self.get(intToValue(0))
	for i := int64(1); i < length; i++ {
		v := o.self.get(intToValue(i))
		if v != nil && v != _undefined {
			o.self.put(intToValue(i-1), v, true)
		} else {
			o.self.delete(intToValue(i-1), true)
		}
	}

	lv := intToValue(length - 1)
	o.self.delete(lv, true)
	o.self.putStr("length", lv, true)

	return first
}

func (r *Runtime) array_isArray(call FunctionCall) Value {
	if o, ok := call.Argument(0).(*Object); ok {
		if isArray(o) {
			return valueTrue
		}
	}
	return valueFalse
}

func (r *Runtime) createArrayProto(val *Object) objectImpl {
	o := &arrayObject{
		baseObject: baseObject{
			class:      classArray,
			val:        val,
			extensible: true,
			prototype:  r.global.ObjectPrototype,
		},
	}
	o.init()

	o._putProp("constructor", r.global.Array, true, false, true)
	o._putProp("pop", r.newNativeFunc(r.arrayproto_pop, nil, "pop", nil, 0), true, false, true)
	o._putProp("push", r.newNativeFunc(r.arrayproto_push, nil, "push", nil, 1), true, false, true)
	o._putProp("join", r.newNativeFunc(r.arrayproto_join, nil, "join", nil, 1), true, false, true)
	o._putProp("toString", r.newNativeFunc(r.arrayproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("toLocaleString", r.newNativeFunc(r.arrayproto_toLocaleString, nil, "toLocaleString", nil, 0), true, false, true)
	o._putProp("concat", r.newNativeFunc(r.arrayproto_concat, nil, "concat", nil, 1), true, false, true)
	o._putProp("reverse", r.newNativeFunc(r.arrayproto_reverse, nil, "reverse", nil, 0), true, false, true)
	o._putProp("shift", r.newNativeFunc(r.arrayproto_shift, nil, "shift", nil, 0), true, false, true)
	o._putProp("slice", r.newNativeFunc(r.arrayproto_slice, nil, "slice", nil, 2), true, false, true)
	o._putProp("sort", r.newNativeFunc(r.arrayproto_sort, nil, "sort", nil, 1), true, false, true)
	o._putProp("splice", r.newNativeFunc(r.arrayproto_splice, nil, "splice", nil, 2), true, false, true)
	o._putProp("unshift", r.newNativeFunc(r.arrayproto_unshift, nil, "unshift", nil, 1), true, false, true)
	o._putProp("indexOf", r.newNativeFunc(r.arrayproto_indexOf, nil, "indexOf", nil, 1), true, false, true)
	o._putProp("lastIndexOf", r.newNativeFunc(r.arrayproto_lastIndexOf, nil, "lastIndexOf", nil, 1), true, false, true)
	o._putProp("every", r.newNativeFunc(r.arrayproto_every, nil, "every", nil, 1), true, false, true)
	o._putProp("some", r.newNativeFunc(r.arrayproto_some, nil, "some", nil, 1), true, false, true)
	o._putProp("forEach", r.newNativeFunc(r.arrayproto_forEach, nil, "forEach", nil, 1), true, false, true)
	o._putProp("map", r.newNativeFunc(r.arrayproto_map, nil, "map", nil, 1), true, false, true)
	o._putProp("filter", r.newNativeFunc(r.arrayproto_filter, nil, "filter", nil, 1), true, false, true)
	o._putProp("reduce", r.newNativeFunc(r.arrayproto_reduce, nil, "reduce", nil, 1), true, false, true)
	o._putProp("reduceRight", r.newNativeFunc(r.arrayproto_reduceRight, nil, "reduceRight", nil, 1), true, false, true)

	return o
}

func (r *Runtime) createArray(val *Object) objectImpl {
	o := r.newNativeFuncConstructObj(val, r.builtin_newArray, "Array", r.global.ArrayPrototype, 1)
	o._putProp("isArray", r.newNativeFunc(r.array_isArray, nil, "isArray", nil, 1), true, false, true)
	return o
}

func (r *Runtime) initArray() {
	//r.global.ArrayPrototype = r.newArray(r.global.ObjectPrototype).val
	//o := r.global.ArrayPrototype.self
	r.global.ArrayPrototype = r.newLazyObject(r.createArrayProto)

	//r.global.Array = r.newNativeFuncConstruct(r.builtin_newArray, "Array", r.global.ArrayPrototype, 1)
	//o = r.global.Array.self
	//o._putProp("isArray", r.newNativeFunc(r.array_isArray, nil, "isArray", nil, 1), true, false, true)
	r.global.Array = r.newLazyObject(r.createArray)

	r.addToGlobal("Array", r.global.Array)
}

type sortable interface {
	sortLen() int64
	sortGet(int64) Value
	swap(int64, int64)
}

type arraySortCtx struct {
	obj     sortable
	compare func(FunctionCall) Value
}

func (ctx *arraySortCtx) sortCompare(x, y Value) int {
	if x == nil && y == nil {
		return 0
	}

	if x == nil {
		return 1
	}

	if y == nil {
		return -1
	}

	if x == _undefined && y == _undefined {
		return 0
	}

	if x == _undefined {
		return 1
	}

	if y == _undefined {
		return -1
	}

	if ctx.compare != nil {
		return int(ctx.compare(FunctionCall{
			This:      _undefined,
			Arguments: []Value{x, y},
		}).ToInteger())
	}
	return strings.Compare(x.String(), y.String())
}

// sort.Interface

func (a *arraySortCtx) Len() int {
	return int(a.obj.sortLen())
}

func (a *arraySortCtx) Less(j, k int) bool {
	return a.sortCompare(a.obj.sortGet(int64(j)), a.obj.sortGet(int64(k))) < 0
}

func (a *arraySortCtx) Swap(j, k int) {
	a.obj.swap(int64(j), int64(k))
}
