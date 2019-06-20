package goja

import (
	"math"
	"reflect"
	"sort"
	"strconv"
)

type sparseArrayItem struct {
	idx   int64
	value Value
}

type sparseArrayObject struct {
	baseObject
	items          []sparseArrayItem
	length         int64
	propValueCount int
	lengthProp     valueProperty
}

func (a *sparseArrayObject) init() {
	a.baseObject.init()
	a.lengthProp.writable = true

	a._put("length", &a.lengthProp)
}

func (a *sparseArrayObject) getLength() Value {
	return intToValue(a.length)
}

func (a *sparseArrayObject) findIdx(idx int64) int {
	return sort.Search(len(a.items), func(i int) bool {
		return a.items[i].idx >= idx
	})
}

func (a *sparseArrayObject) _setLengthInt(l int64, throw bool) bool {
	if l >= 0 && l <= math.MaxUint32 {
		ret := true

		if l <= a.length {
			if a.propValueCount > 0 {
				// Slow path
				for i := len(a.items) - 1; i >= 0; i-- {
					item := a.items[i]
					if item.idx <= l {
						break
					}
					if prop, ok := item.value.(*valueProperty); ok {
						if !prop.configurable {
							l = item.idx + 1
							ret = false
							break
						}
						a.propValueCount--
					}
				}
			}
		}

		idx := a.findIdx(l)

		aa := a.items[idx:]
		for i, _ := range aa {
			aa[i].value = nil
		}
		a.items = a.items[:idx]
		a.length = l
		if !ret {
			a.val.runtime.typeErrorResult(throw, "Cannot redefine property: length")
		}
		return ret
	}
	panic(a.val.runtime.newError(a.val.runtime.global.RangeError, "Invalid array length"))
}

func (a *sparseArrayObject) setLengthInt(l int64, throw bool) bool {
	if l == a.length {
		return true
	}
	if !a.lengthProp.writable {
		a.val.runtime.typeErrorResult(throw, "length is not writable")
		return false
	}
	return a._setLengthInt(l, throw)
}

func (a *sparseArrayObject) setLength(v Value, throw bool) bool {
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

func (a *sparseArrayObject) getIdx(idx int64, origNameStr string, origName Value) (v Value) {
	i := a.findIdx(idx)
	if i < len(a.items) && a.items[i].idx == idx {
		return a.items[i].value
	}

	if a.prototype != nil {
		if origName != nil {
			v = a.prototype.self.getProp(origName)
		} else {
			v = a.prototype.self.getPropStr(origNameStr)
		}
	}
	return
}

func (a *sparseArrayObject) getProp(n Value) Value {
	if idx := toIdx(n); idx >= 0 {
		return a.getIdx(idx, "", n)
	}

	if n.String() == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getProp(n)
}

func (a *sparseArrayObject) getLengthProp() Value {
	a.lengthProp.value = intToValue(a.length)
	return &a.lengthProp
}

func (a *sparseArrayObject) getOwnProp(name string) Value {
	if idx := strToIdx(name); idx >= 0 {
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			return a.items[i].value
		}
		return nil
	}
	if name == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getOwnProp(name)
}

func (a *sparseArrayObject) getPropStr(name string) Value {
	if i := strToIdx(name); i >= 0 {
		return a.getIdx(i, name, nil)
	}
	if name == "length" {
		return a.getLengthProp()
	}
	return a.baseObject.getPropStr(name)
}

func (a *sparseArrayObject) putIdx(idx int64, val Value, throw bool, origNameStr string, origName Value) {
	var prop Value
	i := a.findIdx(idx)
	if i < len(a.items) && a.items[i].idx == idx {
		prop = a.items[i].value
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

		if a.expand() {
			a.items = append(a.items, sparseArrayItem{})
			copy(a.items[i+1:], a.items[i:])
			a.items[i] = sparseArrayItem{
				idx:   idx,
				value: val,
			}
		} else {
			a.val.self.(*arrayObject).putIdx(idx, val, throw, origNameStr, origName)
			return
		}
	} else {
		if prop, ok := prop.(*valueProperty); ok {
			if !prop.isWritable() {
				a.val.runtime.typeErrorResult(throw)
				return
			}
			prop.set(a.val, val)
			return
		} else {
			a.items[i].value = val
		}
	}

}

func (a *sparseArrayObject) put(n Value, val Value, throw bool) {
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

func (a *sparseArrayObject) putStr(name string, val Value, throw bool) {
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

type sparseArrayPropIter struct {
	a         *sparseArrayObject
	recursive bool
	idx       int
}

func (i *sparseArrayPropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.a.items) {
		name := strconv.Itoa(int(i.a.items[i.idx].idx))
		prop := i.a.items[i.idx].value
		i.idx++
		if prop != nil {
			return propIterItem{name: name, value: prop}, i.next
		}
	}

	return i.a.baseObject._enumerate(i.recursive)()
}

func (a *sparseArrayObject) _enumerate(recursive bool) iterNextFunc {
	return (&sparseArrayPropIter{
		a:         a,
		recursive: recursive,
	}).next
}

func (a *sparseArrayObject) enumerate(all, recursive bool) iterNextFunc {
	return (&propFilterIter{
		wrapped: a._enumerate(recursive),
		all:     all,
		seen:    make(map[string]bool),
	}).next
}

func (a *sparseArrayObject) setValues(values []Value) {
	a.items = nil
	for i, val := range values {
		if val != nil {
			a.items = append(a.items, sparseArrayItem{
				idx:   int64(i),
				value: val,
			})
		}
	}
}

func (a *sparseArrayObject) hasOwnProperty(n Value) bool {
	if idx := toIdx(n); idx >= 0 {
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			return a.items[i].value != _undefined
		}
		return false
	} else {
		return a.baseObject.hasOwnProperty(n)
	}
}

func (a *sparseArrayObject) hasOwnPropertyStr(name string) bool {
	if idx := strToIdx(name); idx >= 0 {
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			return a.items[i].value != _undefined
		}
		return false
	} else {
		return a.baseObject.hasOwnPropertyStr(name)
	}
}

func (a *sparseArrayObject) expand() bool {
	if l := len(a.items); l >= 1024 {
		if int(a.items[l-1].idx)/l < 8 {
			//log.Println("Switching sparse->standard")
			ar := &arrayObject{
				baseObject:     a.baseObject,
				length:         a.length,
				propValueCount: a.propValueCount,
			}
			ar.setValuesFromSparse(a.items)
			ar.val.self = ar
			ar.init()
			ar.lengthProp.writable = a.lengthProp.writable
			return false
		}
	}
	return true
}

func (a *sparseArrayObject) defineOwnProperty(n Value, descr propertyDescr, throw bool) bool {
	if idx := toIdx(n); idx >= 0 {
		var existing Value
		i := a.findIdx(idx)
		if i < len(a.items) && a.items[i].idx == idx {
			existing = a.items[i].value
		}
		prop, ok := a.baseObject._defineOwnProperty(n, existing, descr, throw)
		if ok {
			if idx >= a.length {
				if !a.setLengthInt(idx+1, throw) {
					return false
				}
			}
			if i >= len(a.items) || a.items[i].idx != idx {
				if a.expand() {
					a.items = append(a.items, sparseArrayItem{})
					copy(a.items[i+1:], a.items[i:])
					a.items[i] = sparseArrayItem{
						idx:   idx,
						value: prop,
					}
					if idx >= a.length {
						a.length = idx + 1
					}
				} else {
					return a.val.self.defineOwnProperty(n, descr, throw)
				}
			} else {
				a.items[i].value = prop
			}
			if _, ok := prop.(*valueProperty); ok {
				a.propValueCount++
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

func (a *sparseArrayObject) _deleteProp(idx int64, throw bool) bool {
	i := a.findIdx(idx)
	if i < len(a.items) && a.items[i].idx == idx {
		if p, ok := a.items[i].value.(*valueProperty); ok {
			if !p.configurable {
				a.val.runtime.typeErrorResult(throw, "Cannot delete property '%d' of %s", idx, a.val.ToString())
				return false
			}
			a.propValueCount--
		}
		copy(a.items[i:], a.items[i+1:])
		a.items[len(a.items)-1].value = nil
		a.items = a.items[:len(a.items)-1]
	}
	return true
}

func (a *sparseArrayObject) delete(n Value, throw bool) bool {
	if idx := toIdx(n); idx >= 0 {
		return a._deleteProp(idx, throw)
	}
	return a.baseObject.delete(n, throw)
}

func (a *sparseArrayObject) deleteStr(name string, throw bool) bool {
	if idx := strToIdx(name); idx >= 0 {
		return a._deleteProp(idx, throw)
	}
	return a.baseObject.deleteStr(name, throw)
}

func (a *sparseArrayObject) sortLen() int64 {
	if len(a.items) > 0 {
		return a.items[len(a.items)-1].idx + 1
	}

	return 0
}

func (a *sparseArrayObject) sortGet(i int64) Value {
	idx := a.findIdx(i)
	if idx < len(a.items) && a.items[idx].idx == i {
		v := a.items[idx].value
		if p, ok := v.(*valueProperty); ok {
			v = p.get(a.val)
		}
		return v
	}
	return nil
}

func (a *sparseArrayObject) swap(i, j int64) {
	idxI := a.findIdx(i)
	idxJ := a.findIdx(j)

	if idxI < len(a.items) && a.items[idxI].idx == i && idxJ < len(a.items) && a.items[idxJ].idx == j {
		a.items[idxI].value, a.items[idxJ].value = a.items[idxJ].value, a.items[idxI].value
	}
}

func (a *sparseArrayObject) export() interface{} {
	arr := make([]interface{}, a.length)
	for _, item := range a.items {
		if item.value != nil {
			arr[item.idx] = item.value.Export()
		}
	}
	return arr
}

func (a *sparseArrayObject) exportType() reflect.Type {
	return reflectTypeArray
}
