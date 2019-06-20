package goja

import (
	"math"
	"reflect"
	"strconv"
)

type arrayObject struct {
	baseObject
	values         []Value
	length         int64
	objCount       int64
	propValueCount int
	lengthProp     valueProperty
}

func (a *arrayObject) init() {
	a.baseObject.init()
	a.lengthProp.writable = true

	a._put("length", &a.lengthProp)
}

func (a *arrayObject) getLength() Value {
	return intToValue(a.length)
}

func (a *arrayObject) _setLengthInt(l int64, throw bool) bool {
	if l >= 0 && l <= math.MaxUint32 {
		ret := true
		if l <= a.length {
			if a.propValueCount > 0 {
				// Slow path
				var s int64
				if a.length < int64(len(a.values)) {
					s = a.length - 1
				} else {
					s = int64(len(a.values)) - 1
				}
				for i := s; i >= l; i-- {
					if prop, ok := a.values[i].(*valueProperty); ok {
						if !prop.configurable {
							l = i + 1
							ret = false
							break
						}
						a.propValueCount--
					}
				}
			}
		}
		if l <= int64(len(a.values)) {
			if l >= 16 && l < int64(cap(a.values))>>2 {
				ar := make([]Value, l)
				copy(ar, a.values)
				a.values = ar
			} else {
				ar := a.values[l:len(a.values)]
				for i, _ := range ar {
					ar[i] = nil
				}
				a.values = a.values[:l]
			}
		}
		a.length = l
		if !ret {
			a.val.runtime.typeErrorResult(throw, "Cannot redefine property: length")
		}
		return ret
	}
	panic(a.val.runtime.newError(a.val.runtime.global.RangeError, "Invalid array length"))
}

func (a *arrayObject) setLengthInt(l int64, throw bool) bool {
	if l == a.length {
		return true
	}
	if !a.lengthProp.writable {
		a.val.runtime.typeErrorResult(throw, "length is not writable")
		return false
	}
	return a._setLengthInt(l, throw)
}

func (a *arrayObject) setLength(v Value, throw bool) bool {
	l, ok := toIntIgnoreNegZero(v)
	if ok && l == a.length {
		return true
	}
	if !a.lengthProp.writable {
		a.val.runtime.typeErrorResult(throw, "length is not writable")
		return false
	}
	if ok {
		return a._setLengthInt(l, throw)
	}
	panic(a.val.runtime.newError(a.val.runtime.global.RangeError, "Invalid array length"))
}

func (a *arrayObject) getIdx(idx int64, origNameStr string, origName Value) (v Value) {
	if idx >= 0 && idx < int64(len(a.values)) {
		v = a.values[idx]
	}
	if v == nil && a.prototype != nil {
		if origName != nil {
			v = a.prototype.self.getProp(origName)
		} else {
			v = a.prototype.self.getPropStr(origNameStr)
		}
	}
	return
}

func (a *arrayObject) sortLen() int64 {
	return int64(len(a.values))
}

func (a *arrayObject) sortGet(i int64) Value {
	v := a.values[i]
	if p, ok := v.(*valueProperty); ok {
		v = p.get(a.val)
	}
	return v
}

func (a *arrayObject) swap(i, j int64) {
	a.values[i], a.values[j] = a.values[j], a.values[i]
}

func toIdx(v Value) (idx int64) {
	idx = -1
	if idxVal, ok1 := v.(valueInt); ok1 {
		idx = int64(idxVal)
	} else {
		if i, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			idx = i
		}
	}
	if idx >= 0 && idx < math.MaxUint32 {
		return
	}
	return -1
}

func strToIdx(s string) (idx int64) {
	idx = -1
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		idx = i
	}

	if idx >= 0 && idx < math.MaxUint32 {
		return
	}
	return -1
}

func (a *arrayObject) getProp(n Value) Value {
	if idx := toIdx(n); idx >= 0 {
		return a.getIdx(idx, "", n)
	}

	if n.String() == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getProp(n)
}

func (a *arrayObject) getLengthProp() Value {
	a.lengthProp.value = intToValue(a.length)
	return &a.lengthProp
}

func (a *arrayObject) getPropStr(name string) Value {
	if i := strToIdx(name); i >= 0 {
		return a.getIdx(i, name, nil)
	}
	if name == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getPropStr(name)
}

func (a *arrayObject) getOwnProp(name string) Value {
	if i := strToIdx(name); i >= 0 {
		if i >= 0 && i < int64(len(a.values)) {
			return a.values[i]
		}
	}
	if name == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getOwnProp(name)
}

func (a *arrayObject) putIdx(idx int64, val Value, throw bool, origNameStr string, origName Value) {
	var prop Value
	if idx < int64(len(a.values)) {
		prop = a.values[idx]
	}

	if prop == nil {
		if a.prototype != nil {
			var pprop Value
			if origName != nil {
				pprop = a.prototype.self.getProp(origName)
			} else {
				pprop = a.prototype.self.getPropStr(origNameStr)
			}
			if pprop, ok := pprop.(*valueProperty); ok {
				if !pprop.isWritable() {
					a.val.runtime.typeErrorResult(throw)
					return
				}
				if pprop.accessor {
					pprop.set(a.val, val)
					return
				}
			}
		}

		if !a.extensible {
			a.val.runtime.typeErrorResult(throw)
			return
		}
		if idx >= a.length {
			if !a.setLengthInt(idx+1, throw) {
				return
			}
		}
		if idx >= int64(len(a.values)) {
			if !a.expand(idx) {
				a.val.self.(*sparseArrayObject).putIdx(idx, val, throw, origNameStr, origName)
				return
			}
		}
	} else {
		if prop, ok := prop.(*valueProperty); ok {
			if !prop.isWritable() {
				a.val.runtime.typeErrorResult(throw)
				return
			}
			prop.set(a.val, val)
			return
		}
	}

	a.values[idx] = val
	a.objCount++
}

func (a *arrayObject) put(n Value, val Value, throw bool) {
	if idx := toIdx(n); idx >= 0 {
		a.putIdx(idx, val, throw, "", n)
	} else {
		if n.String() == "length" {
			a.setLength(val, throw)
		} else {
			a.baseObject.put(n, val, throw)
		}
	}
}

func (a *arrayObject) putStr(name string, val Value, throw bool) {
	if idx := strToIdx(name); idx >= 0 {
		a.putIdx(idx, val, throw, name, nil)
	} else {
		if name == "length" {
			a.setLength(val, throw)
		} else {
			a.baseObject.putStr(name, val, throw)
		}
	}
}

type arrayPropIter struct {
	a         *arrayObject
	recursive bool
	idx       int
}

func (i *arrayPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.a.values) {
		name := strconv.Itoa(i.idx)
		prop := i.a.values[i.idx]
		i.idx++
		if prop != nil {
			return propIterItem{name: name, value: prop}, i.next
		}
	}

	return i.a.baseObject._enumerate(i.recursive)()
}

func (a *arrayObject) _enumerate(recursive bool) iterNextFunc {
	return (&arrayPropIter{
		a:         a,
		recursive: recursive,
	}).next
}

func (a *arrayObject) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: a._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (a *arrayObject) hasOwnProperty(n Value) bool {
	if idx := toIdx(n); idx >= 0 {
		return idx < int64(len(a.values)) && a.values[idx] != nil && a.values[idx] != _undefined
	} else {
		return a.baseObject.hasOwnProperty(n)
	}
}

func (a *arrayObject) hasOwnPropertyStr(name string) bool {
	if idx := strToIdx(name); idx >= 0 {
		return idx < int64(len(a.values)) && a.values[idx] != nil && a.values[idx] != _undefined
	} else {
		return a.baseObject.hasOwnPropertyStr(name)
	}
}

func (a *arrayObject) expand(idx int64) bool {
	targetLen := idx + 1
	if targetLen > int64(len(a.values)) {
		if targetLen < int64(cap(a.values)) {
			a.values = a.values[:targetLen]
		} else {
			if idx > 4096 && (a.objCount == 0 || idx/a.objCount > 10) {
				//log.Println("Switching standard->sparse")
				sa := &sparseArrayObject{
					baseObject:     a.baseObject,
					length:         a.length,
					propValueCount: a.propValueCount,
				}
				sa.setValues(a.values)
				sa.val.self = sa
				sa.init()
				sa.lengthProp.writable = a.lengthProp.writable
				return false
			} else {
				// Use the same algorithm as in runtime.growSlice
				newcap := int64(cap(a.values))
				doublecap := newcap + newcap
				if targetLen > doublecap {
					newcap = targetLen
				} else {
					if len(a.values) < 1024 {
						newcap = doublecap
					} else {
						for newcap < targetLen {
							newcap += newcap / 4
						}
					}
				}
				newValues := make([]Value, targetLen, newcap)
				copy(newValues, a.values)
				a.values = newValues
			}
		}
	}
	return true
}

func (r *Runtime) defineArrayLength(prop *valueProperty, descr propertyDescr, setter func(Value, bool) bool, throw bool) bool {
	ret := true

	if descr.Configurable == FLAG_TRUE || descr.Enumerable == FLAG_TRUE || descr.Getter != nil || descr.Setter != nil {
		ret = false
		goto Reject
	}

	if newLen := descr.Value; newLen != nil {
		ret = setter(newLen, false)
	} else {
		ret = true
	}

	if descr.Writable != FLAG_NOT_SET {
		w := descr.Writable.Bool()
		if prop.writable {
			prop.writable = w
		} else {
			if w {
				ret = false
				goto Reject
			}
		}
	}

Reject:
	if !ret {
		r.typeErrorResult(throw, "Cannot redefine property: length")
	}

	return ret
}

func (a *arrayObject) defineOwnProperty(n Value, descr propertyDescr, throw bool) bool {
	if idx := toIdx(n); idx >= 0 {
		var existing Value
		if idx < int64(len(a.values)) {
			existing = a.values[idx]
		}
		prop, ok := a.baseObject._defineOwnProperty(n, existing, descr, throw)
		if ok {
			if idx >= a.length {
				if !a.setLengthInt(idx+1, throw) {
					return false
				}
			}
			if a.expand(idx) {
				a.values[idx] = prop
				a.objCount++
				if _, ok := prop.(*valueProperty); ok {
					a.propValueCount++
				}
			} else {
				a.val.self.(*sparseArrayObject).putIdx(idx, prop, throw, "", nil)
			}
		}
		return ok
	} else {
		if n.String() == "length" {
			return a.val.runtime.defineArrayLength(&a.lengthProp, descr, a.setLength, throw)
		}
		return a.baseObject.defineOwnProperty(n, descr, throw)
	}
}

func (a *arrayObject) _deleteProp(idx int64, throw bool) bool {
	if idx < int64(len(a.values)) {
		if v := a.values[idx]; v != nil {
			if p, ok := v.(*valueProperty); ok {
				if !p.configurable {
					a.val.runtime.typeErrorResult(throw, "Cannot delete property '%d' of %s", idx, a.val.ToString())
					return false
				}
				a.propValueCount--
			}
			a.values[idx] = nil
			a.objCount--
		}
	}
	return true
}

func (a *arrayObject) delete(n Value, throw bool) bool {
	if idx := toIdx(n); idx >= 0 {
		return a._deleteProp(idx, throw)
	}
	return a.baseObject.delete(n, throw)
}

func (a *arrayObject) deleteStr(name string, throw bool) bool {
	if idx := strToIdx(name); idx >= 0 {
		return a._deleteProp(idx, throw)
	}
	return a.baseObject.deleteStr(name, throw)
}

func (a *arrayObject) export() interface{} {
	arr := make([]interface{}, a.length)
	for i, v := range a.values {
		if v != nil {
			arr[i] = v.Export()
		}
	}

	return arr
}

func (a *arrayObject) exportType() reflect.Type {
	return reflectTypeArray
}

func (a *arrayObject) setValuesFromSparse(items []sparseArrayItem) {
	a.values = make([]Value, int(items[len(items)-1].idx+1))
	for _, item := range items {
		a.values[item.idx] = item.value
	}
	a.objCount = int64(len(items))
}
