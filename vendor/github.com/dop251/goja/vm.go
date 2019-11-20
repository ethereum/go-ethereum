package goja

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	maxInt = 1 << 53
)

type valueStack []Value

type stash struct {
	values    valueStack
	extraArgs valueStack
	names     map[string]uint32
	obj       objectImpl

	outer *stash
}

type context struct {
	prg      *Program
	funcName string
	stash    *stash
	pc, sb   int
	args     int
}

type iterStackItem struct {
	val Value
	f   iterNextFunc
}

type ref interface {
	get() Value
	set(Value)
	refname() string
}

type stashRef struct {
	v *Value
	n string
}

func (r stashRef) get() Value {
	return *r.v
}

func (r *stashRef) set(v Value) {
	*r.v = v
}

func (r *stashRef) refname() string {
	return r.n
}

type objRef struct {
	base   objectImpl
	name   string
	strict bool
}

func (r *objRef) get() Value {
	return r.base.getStr(r.name)
}

func (r *objRef) set(v Value) {
	r.base.putStr(r.name, v, r.strict)
}

func (r *objRef) refname() string {
	return r.name
}

type unresolvedRef struct {
	runtime *Runtime
	name    string
}

func (r *unresolvedRef) get() Value {
	r.runtime.throwReferenceError(r.name)
	panic("Unreachable")
}

func (r *unresolvedRef) set(v Value) {
	r.get()
}

func (r *unresolvedRef) refname() string {
	return r.name
}

type vm struct {
	r            *Runtime
	prg          *Program
	funcName     string
	pc           int
	stack        valueStack
	sp, sb, args int

	stash     *stash
	callStack []context
	iterStack []iterStackItem
	refStack  []ref

	stashAllocs int
	halt        bool

	interrupted   uint32
	interruptVal  interface{}
	interruptLock sync.Mutex
}

type instruction interface {
	exec(*vm)
}

func intToValue(i int64) Value {
	if i >= -maxInt && i <= maxInt {
		if i >= -128 && i <= 127 {
			return intCache[i+128]
		}
		return valueInt(i)
	}
	return valueFloat(float64(i))
}

func floatToInt(f float64) (result int64, ok bool) {
	if (f != 0 || !math.Signbit(f)) && !math.IsInf(f, 0) && f == math.Trunc(f) && f >= -maxInt && f <= maxInt {
		return int64(f), true
	}
	return 0, false
}

func floatToValue(f float64) (result Value) {
	if i, ok := floatToInt(f); ok {
		return intToValue(i)
	}
	switch {
	case f == 0:
		return _negativeZero
	case math.IsNaN(f):
		return _NaN
	case math.IsInf(f, 1):
		return _positiveInf
	case math.IsInf(f, -1):
		return _negativeInf
	}
	return valueFloat(f)
}

func toInt(v Value) (int64, bool) {
	num := v.ToNumber()
	if i, ok := num.assertInt(); ok {
		return i, true
	}
	if f, ok := num.assertFloat(); ok {
		if i, ok := floatToInt(f); ok {
			return i, true
		}
	}
	return 0, false
}

func toIntIgnoreNegZero(v Value) (int64, bool) {
	num := v.ToNumber()
	if i, ok := num.assertInt(); ok {
		return i, true
	}
	if f, ok := num.assertFloat(); ok {
		if v == _negativeZero {
			return 0, true
		}
		if i, ok := floatToInt(f); ok {
			return i, true
		}
	}
	return 0, false
}

func (s *valueStack) expand(idx int) {
	if idx < len(*s) {
		return
	}

	if idx < cap(*s) {
		*s = (*s)[:idx+1]
	} else {
		n := make([]Value, idx+1, (idx+1)<<1)
		copy(n, *s)
		*s = n
	}
}

func (s *stash) put(name string, v Value) bool {
	if s.obj != nil {
		if found := s.obj.getStr(name); found != nil {
			s.obj.putStr(name, v, false)
			return true
		}
		return false
	} else {
		if idx, found := s.names[name]; found {
			s.values.expand(int(idx))
			s.values[idx] = v
			return true
		}
		return false
	}
}

func (s *stash) putByIdx(idx uint32, v Value) {
	if s.obj != nil {
		panic("Attempt to put by idx into an object scope")
	}
	s.values.expand(int(idx))
	s.values[idx] = v
}

func (s *stash) getByIdx(idx uint32) Value {
	if int(idx) < len(s.values) {
		return s.values[idx]
	}
	return _undefined
}

func (s *stash) getByName(name string, vm *vm) (v Value, exists bool) {
	if s.obj != nil {
		v = s.obj.getStr(name)
		if v == nil {
			return nil, false
			//return valueUnresolved{r: vm.r, ref: name}, false
		}
		return v, true
	}
	if idx, exists := s.names[name]; exists {
		return s.values[idx], true
	}
	return nil, false
	//return valueUnresolved{r: vm.r, ref: name}, false
}

func (s *stash) createBinding(name string) {
	if s.names == nil {
		s.names = make(map[string]uint32)
	}
	if _, exists := s.names[name]; !exists {
		s.names[name] = uint32(len(s.names))
		s.values = append(s.values, _undefined)
	}
}

func (s *stash) deleteBinding(name string) bool {
	if s.obj != nil {
		return s.obj.deleteStr(name, false)
	}
	if idx, found := s.names[name]; found {
		s.values[idx] = nil
		delete(s.names, name)
		return true
	}
	return false
}

func (vm *vm) newStash() {
	vm.stash = &stash{
		outer: vm.stash,
	}
	vm.stashAllocs++
}

func (vm *vm) init() {
}

func (vm *vm) run() {
	vm.halt = false
	interrupted := false
	ticks := 0
	for !vm.halt {
		if interrupted = atomic.LoadUint32(&vm.interrupted) != 0; interrupted {
			break
		}
		vm.prg.code[vm.pc].exec(vm)
		ticks++
		if ticks > 10000 {
			runtime.Gosched()
			ticks = 0
		}
	}

	if interrupted {
		vm.interruptLock.Lock()
		v := &InterruptedError{
			iface: vm.interruptVal,
		}
		atomic.StoreUint32(&vm.interrupted, 0)
		vm.interruptVal = nil
		vm.interruptLock.Unlock()
		panic(v)
	}
}

func (vm *vm) Interrupt(v interface{}) {
	vm.interruptLock.Lock()
	vm.interruptVal = v
	atomic.StoreUint32(&vm.interrupted, 1)
	vm.interruptLock.Unlock()
}

func (vm *vm) captureStack(stack []stackFrame, ctxOffset int) []stackFrame {
	// Unroll the context stack
	stack = append(stack, stackFrame{prg: vm.prg, pc: vm.pc, funcName: vm.funcName})
	for i := len(vm.callStack) - 1; i > ctxOffset-1; i-- {
		if vm.callStack[i].pc != -1 {
			stack = append(stack, stackFrame{prg: vm.callStack[i].prg, pc: vm.callStack[i].pc - 1, funcName: vm.callStack[i].funcName})
		}
	}
	return stack
}

func (vm *vm) try(f func()) (ex *Exception) {
	var ctx context
	vm.saveCtx(&ctx)

	ctxOffset := len(vm.callStack)
	sp := vm.sp
	iterLen := len(vm.iterStack)
	refLen := len(vm.refStack)

	defer func() {
		if x := recover(); x != nil {
			defer func() {
				vm.callStack = vm.callStack[:ctxOffset]
				vm.restoreCtx(&ctx)
				vm.sp = sp

				// Restore other stacks
				iterTail := vm.iterStack[iterLen:]
				for i, _ := range iterTail {
					iterTail[i] = iterStackItem{}
				}
				vm.iterStack = vm.iterStack[:iterLen]
				refTail := vm.refStack[refLen:]
				for i, _ := range refTail {
					refTail[i] = nil
				}
				vm.refStack = vm.refStack[:refLen]
			}()
			switch x1 := x.(type) {
			case Value:
				ex = &Exception{
					val: x1,
				}
			case *InterruptedError:
				x1.stack = vm.captureStack(x1.stack, ctxOffset)
				panic(x1)
			case *Exception:
				ex = x1
			default:
				/*
					if vm.prg != nil {
						vm.prg.dumpCode(log.Printf)
					}
					log.Print("Stack: ", string(debug.Stack()))
					panic(fmt.Errorf("Panic at %d: %v", vm.pc, x))
				*/
				panic(x)
			}
			ex.stack = vm.captureStack(ex.stack, ctxOffset)
		}
	}()

	f()
	return
}

func (vm *vm) runTry() (ex *Exception) {
	return vm.try(vm.run)
}

func (vm *vm) push(v Value) {
	vm.stack.expand(vm.sp)
	vm.stack[vm.sp] = v
	vm.sp++
}

func (vm *vm) pop() Value {
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *vm) peek() Value {
	return vm.stack[vm.sp-1]
}

func (vm *vm) saveCtx(ctx *context) {
	ctx.prg = vm.prg
	ctx.funcName = vm.funcName
	ctx.stash = vm.stash
	ctx.pc = vm.pc
	ctx.sb = vm.sb
	ctx.args = vm.args
}

func (vm *vm) pushCtx() {
	/*
		vm.ctxStack = append(vm.ctxStack, context{
			prg: vm.prg,
			stash: vm.stash,
			pc: vm.pc,
			sb: vm.sb,
			args: vm.args,
		})*/
	vm.callStack = append(vm.callStack, context{})
	vm.saveCtx(&vm.callStack[len(vm.callStack)-1])
}

func (vm *vm) restoreCtx(ctx *context) {
	vm.prg = ctx.prg
	vm.funcName = ctx.funcName
	vm.pc = ctx.pc
	vm.stash = ctx.stash
	vm.sb = ctx.sb
	vm.args = ctx.args
}

func (vm *vm) popCtx() {
	l := len(vm.callStack) - 1
	vm.prg = vm.callStack[l].prg
	vm.callStack[l].prg = nil
	vm.funcName = vm.callStack[l].funcName
	vm.pc = vm.callStack[l].pc
	vm.stash = vm.callStack[l].stash
	vm.callStack[l].stash = nil
	vm.sb = vm.callStack[l].sb
	vm.args = vm.callStack[l].args

	vm.callStack = vm.callStack[:l]
}

func (r *Runtime) toObject(v Value, args ...interface{}) *Object {
	//r.checkResolveable(v)
	if obj, ok := v.(*Object); ok {
		return obj
	}
	if len(args) > 0 {
		r.typeErrorResult(true, args)
	} else {
		r.typeErrorResult(true, "Value is not an object: %s", v.ToString())
	}
	panic("Unreachable")
}

func (r *Runtime) toCallee(v Value) *Object {
	if obj, ok := v.(*Object); ok {
		return obj
	}
	switch unresolved := v.(type) {
	case valueUnresolved:
		unresolved.throw()
		panic("Unreachable")
	case memberUnresolved:
		r.typeErrorResult(true, "Object has no member '%s'", unresolved.ref)
		panic("Unreachable")
	}
	r.typeErrorResult(true, "Value is not an object: %s", v.ToString())
	panic("Unreachable")
}

type _newStash struct{}

var newStash _newStash

func (_newStash) exec(vm *vm) {
	vm.newStash()
	vm.pc++
}

type _noop struct{}

var noop _noop

func (_noop) exec(vm *vm) {
	vm.pc++
}

type loadVal uint32

func (l loadVal) exec(vm *vm) {
	vm.push(vm.prg.values[l])
	vm.pc++
}

type loadVal1 uint32

func (l *loadVal1) exec(vm *vm) {
	vm.push(vm.prg.values[*l])
	vm.pc++
}

type _loadUndef struct{}

var loadUndef _loadUndef

func (_loadUndef) exec(vm *vm) {
	vm.push(_undefined)
	vm.pc++
}

type _loadNil struct{}

var loadNil _loadNil

func (_loadNil) exec(vm *vm) {
	vm.push(nil)
	vm.pc++
}

type _loadGlobalObject struct{}

var loadGlobalObject _loadGlobalObject

func (_loadGlobalObject) exec(vm *vm) {
	vm.push(vm.r.globalObject)
	vm.pc++
}

type loadStack int

func (l loadStack) exec(vm *vm) {
	// l < 0 -- arg<-l-1>
	// l > 0 -- var<l-1>
	// l == 0 -- this

	if l < 0 {
		arg := int(-l)
		if arg > vm.args {
			vm.push(_undefined)
		} else {
			vm.push(vm.stack[vm.sb+arg])
		}
	} else if l > 0 {
		vm.push(vm.stack[vm.sb+vm.args+int(l)])
	} else {
		vm.push(vm.stack[vm.sb])
	}
	vm.pc++
}

type _loadCallee struct{}

var loadCallee _loadCallee

func (_loadCallee) exec(vm *vm) {
	vm.push(vm.stack[vm.sb-1])
	vm.pc++
}

func (vm *vm) storeStack(s int) {
	// l < 0 -- arg<-l-1>
	// l > 0 -- var<l-1>
	// l == 0 -- this

	if s < 0 {
		vm.stack[vm.sb-s] = vm.stack[vm.sp-1]
	} else if s > 0 {
		vm.stack[vm.sb+vm.args+s] = vm.stack[vm.sp-1]
	} else {
		panic("Attempt to modify this")
	}
	vm.pc++
}

type storeStack int

func (s storeStack) exec(vm *vm) {
	vm.storeStack(int(s))
}

type storeStackP int

func (s storeStackP) exec(vm *vm) {
	vm.storeStack(int(s))
	vm.sp--
}

type _toNumber struct{}

var toNumber _toNumber

func (_toNumber) exec(vm *vm) {
	vm.stack[vm.sp-1] = vm.stack[vm.sp-1].ToNumber()
	vm.pc++
}

type _add struct{}

var add _add

func (_add) exec(vm *vm) {
	right := vm.stack[vm.sp-1]
	left := vm.stack[vm.sp-2]

	if o, ok := left.(*Object); ok {
		left = o.self.toPrimitive()
	}

	if o, ok := right.(*Object); ok {
		right = o.self.toPrimitive()
	}

	var ret Value

	leftString, isLeftString := left.assertString()
	rightString, isRightString := right.assertString()

	if isLeftString || isRightString {
		if !isLeftString {
			leftString = left.ToString()
		}
		if !isRightString {
			rightString = right.ToString()
		}
		ret = leftString.concat(rightString)
	} else {
		if leftInt, ok := left.assertInt(); ok {
			if rightInt, ok := right.assertInt(); ok {
				ret = intToValue(int64(leftInt) + int64(rightInt))
			} else {
				ret = floatToValue(float64(leftInt) + right.ToFloat())
			}
		} else {
			ret = floatToValue(left.ToFloat() + right.ToFloat())
		}
	}

	vm.stack[vm.sp-2] = ret
	vm.sp--
	vm.pc++
}

type _sub struct{}

var sub _sub

func (_sub) exec(vm *vm) {
	right := vm.stack[vm.sp-1]
	left := vm.stack[vm.sp-2]

	var result Value

	if left, ok := left.assertInt(); ok {
		if right, ok := right.assertInt(); ok {
			result = intToValue(left - right)
			goto end
		}
	}

	result = floatToValue(left.ToFloat() - right.ToFloat())
end:
	vm.sp--
	vm.stack[vm.sp-1] = result
	vm.pc++
}

type _mul struct{}

var mul _mul

func (_mul) exec(vm *vm) {
	left := vm.stack[vm.sp-2]
	right := vm.stack[vm.sp-1]

	var result Value

	if left, ok := toInt(left); ok {
		if right, ok := toInt(right); ok {
			if left == 0 && right == -1 || left == -1 && right == 0 {
				result = _negativeZero
				goto end
			}
			res := left * right
			// check for overflow
			if left == 0 || right == 0 || res/left == right {
				result = intToValue(res)
				goto end
			}

		}
	}

	result = floatToValue(left.ToFloat() * right.ToFloat())

end:
	vm.sp--
	vm.stack[vm.sp-1] = result
	vm.pc++
}

type _div struct{}

var div _div

func (_div) exec(vm *vm) {
	left := vm.stack[vm.sp-2].ToFloat()
	right := vm.stack[vm.sp-1].ToFloat()

	var result Value

	if math.IsNaN(left) || math.IsNaN(right) {
		result = _NaN
		goto end
	}
	if math.IsInf(left, 0) && math.IsInf(right, 0) {
		result = _NaN
		goto end
	}
	if left == 0 && right == 0 {
		result = _NaN
		goto end
	}

	if math.IsInf(left, 0) {
		if math.Signbit(left) == math.Signbit(right) {
			result = _positiveInf
			goto end
		} else {
			result = _negativeInf
			goto end
		}
	}
	if math.IsInf(right, 0) {
		if math.Signbit(left) == math.Signbit(right) {
			result = _positiveZero
			goto end
		} else {
			result = _negativeZero
			goto end
		}
	}
	if right == 0 {
		if math.Signbit(left) == math.Signbit(right) {
			result = _positiveInf
			goto end
		} else {
			result = _negativeInf
			goto end
		}
	}

	result = floatToValue(left / right)

end:
	vm.sp--
	vm.stack[vm.sp-1] = result
	vm.pc++
}

type _mod struct{}

var mod _mod

func (_mod) exec(vm *vm) {
	left := vm.stack[vm.sp-2]
	right := vm.stack[vm.sp-1]

	var result Value

	if leftInt, ok := toInt(left); ok {
		if rightInt, ok := toInt(right); ok {
			if rightInt == 0 {
				result = _NaN
				goto end
			}
			r := leftInt % rightInt
			if r == 0 && leftInt < 0 {
				result = _negativeZero
			} else {
				result = intToValue(leftInt % rightInt)
			}
			goto end
		}
	}

	result = floatToValue(math.Mod(left.ToFloat(), right.ToFloat()))
end:
	vm.sp--
	vm.stack[vm.sp-1] = result
	vm.pc++
}

type _neg struct{}

var neg _neg

func (_neg) exec(vm *vm) {
	operand := vm.stack[vm.sp-1]

	var result Value

	if i, ok := toInt(operand); ok {
		if i == 0 {
			result = _negativeZero
		} else {
			result = valueInt(-i)
		}
	} else {
		f := operand.ToFloat()
		if !math.IsNaN(f) {
			f = -f
		}
		result = valueFloat(f)
	}

	vm.stack[vm.sp-1] = result
	vm.pc++
}

type _plus struct{}

var plus _plus

func (_plus) exec(vm *vm) {
	vm.stack[vm.sp-1] = vm.stack[vm.sp-1].ToNumber()
	vm.pc++
}

type _inc struct{}

var inc _inc

func (_inc) exec(vm *vm) {
	v := vm.stack[vm.sp-1]

	if i, ok := toInt(v); ok {
		v = intToValue(i + 1)
		goto end
	}

	v = valueFloat(v.ToFloat() + 1)

end:
	vm.stack[vm.sp-1] = v
	vm.pc++
}

type _dec struct{}

var dec _dec

func (_dec) exec(vm *vm) {
	v := vm.stack[vm.sp-1]

	if i, ok := toInt(v); ok {
		v = intToValue(i - 1)
		goto end
	}

	v = valueFloat(v.ToFloat() - 1)

end:
	vm.stack[vm.sp-1] = v
	vm.pc++
}

type _and struct{}

var and _and

func (_and) exec(vm *vm) {
	left := toInt32(vm.stack[vm.sp-2])
	right := toInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-2] = intToValue(int64(left & right))
	vm.sp--
	vm.pc++
}

type _or struct{}

var or _or

func (_or) exec(vm *vm) {
	left := toInt32(vm.stack[vm.sp-2])
	right := toInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-2] = intToValue(int64(left | right))
	vm.sp--
	vm.pc++
}

type _xor struct{}

var xor _xor

func (_xor) exec(vm *vm) {
	left := toInt32(vm.stack[vm.sp-2])
	right := toInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-2] = intToValue(int64(left ^ right))
	vm.sp--
	vm.pc++
}

type _bnot struct{}

var bnot _bnot

func (_bnot) exec(vm *vm) {
	op := toInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-1] = intToValue(int64(^op))
	vm.pc++
}

type _sal struct{}

var sal _sal

func (_sal) exec(vm *vm) {
	left := toInt32(vm.stack[vm.sp-2])
	right := toUInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-2] = intToValue(int64(left << (right & 0x1F)))
	vm.sp--
	vm.pc++
}

type _sar struct{}

var sar _sar

func (_sar) exec(vm *vm) {
	left := toInt32(vm.stack[vm.sp-2])
	right := toUInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-2] = intToValue(int64(left >> (right & 0x1F)))
	vm.sp--
	vm.pc++
}

type _shr struct{}

var shr _shr

func (_shr) exec(vm *vm) {
	left := toUInt32(vm.stack[vm.sp-2])
	right := toUInt32(vm.stack[vm.sp-1])
	vm.stack[vm.sp-2] = intToValue(int64(left >> (right & 0x1F)))
	vm.sp--
	vm.pc++
}

type _halt struct{}

var halt _halt

func (_halt) exec(vm *vm) {
	vm.halt = true
	vm.pc++
}

type jump int32

func (j jump) exec(vm *vm) {
	vm.pc += int(j)
}

type _setElem struct{}

var setElem _setElem

func (_setElem) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-3])
	propName := vm.stack[vm.sp-2]
	val := vm.stack[vm.sp-1]

	obj.self.put(propName, val, false)

	vm.sp -= 2
	vm.stack[vm.sp-1] = val
	vm.pc++
}

type _setElemStrict struct{}

var setElemStrict _setElemStrict

func (_setElemStrict) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-3])
	propName := vm.stack[vm.sp-2]
	val := vm.stack[vm.sp-1]

	obj.self.put(propName, val, true)

	vm.sp -= 2
	vm.stack[vm.sp-1] = val
	vm.pc++
}

type _deleteElem struct{}

var deleteElem _deleteElem

func (_deleteElem) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-2])
	propName := vm.stack[vm.sp-1]
	if !obj.self.hasProperty(propName) || obj.self.delete(propName, false) {
		vm.stack[vm.sp-2] = valueTrue
	} else {
		vm.stack[vm.sp-2] = valueFalse
	}
	vm.sp--
	vm.pc++
}

type _deleteElemStrict struct{}

var deleteElemStrict _deleteElemStrict

func (_deleteElemStrict) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-2])
	propName := vm.stack[vm.sp-1]
	obj.self.delete(propName, true)
	vm.stack[vm.sp-2] = valueTrue
	vm.sp--
	vm.pc++
}

type deleteProp string

func (d deleteProp) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-1])
	if !obj.self.hasPropertyStr(string(d)) || obj.self.deleteStr(string(d), false) {
		vm.stack[vm.sp-1] = valueTrue
	} else {
		vm.stack[vm.sp-1] = valueFalse
	}
	vm.pc++
}

type deletePropStrict string

func (d deletePropStrict) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-1])
	obj.self.deleteStr(string(d), true)
	vm.stack[vm.sp-1] = valueTrue
	vm.pc++
}

type setProp string

func (p setProp) exec(vm *vm) {
	val := vm.stack[vm.sp-1]

	vm.r.toObject(vm.stack[vm.sp-2]).self.putStr(string(p), val, false)
	vm.stack[vm.sp-2] = val
	vm.sp--
	vm.pc++
}

type setPropStrict string

func (p setPropStrict) exec(vm *vm) {
	obj := vm.stack[vm.sp-2]
	val := vm.stack[vm.sp-1]

	obj1 := vm.r.toObject(obj)
	obj1.self.putStr(string(p), val, true)
	vm.stack[vm.sp-2] = val
	vm.sp--
	vm.pc++
}

type setProp1 string

func (p setProp1) exec(vm *vm) {
	vm.r.toObject(vm.stack[vm.sp-2]).self._putProp(string(p), vm.stack[vm.sp-1], true, true, true)

	vm.sp--
	vm.pc++
}

type _setProto struct{}

var setProto _setProto

func (_setProto) exec(vm *vm) {
	vm.r.toObject(vm.stack[vm.sp-2]).self.putStr("__proto__", vm.stack[vm.sp-1], true)

	vm.sp--
	vm.pc++
}

type setPropGetter string

func (s setPropGetter) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-2])
	val := vm.stack[vm.sp-1]

	descr := propertyDescr{
		Getter:       val,
		Configurable: FLAG_TRUE,
		Enumerable:   FLAG_TRUE,
	}

	obj.self.defineOwnProperty(newStringValue(string(s)), descr, false)

	vm.sp--
	vm.pc++
}

type setPropSetter string

func (s setPropSetter) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-2])
	val := vm.stack[vm.sp-1]

	descr := propertyDescr{
		Setter:       val,
		Configurable: FLAG_TRUE,
		Enumerable:   FLAG_TRUE,
	}

	obj.self.defineOwnProperty(newStringValue(string(s)), descr, false)

	vm.sp--
	vm.pc++
}

type getProp string

func (g getProp) exec(vm *vm) {
	v := vm.stack[vm.sp-1]
	obj := v.baseObject(vm.r)
	if obj == nil {
		vm.r.typeErrorResult(true, "Cannot read property '%s' of undefined", g)
	}
	prop := obj.self.getPropStr(string(g))
	if prop1, ok := prop.(*valueProperty); ok {
		vm.stack[vm.sp-1] = prop1.get(v)
	} else {
		if prop == nil {
			prop = _undefined
		}
		vm.stack[vm.sp-1] = prop
	}

	vm.pc++
}

type getPropCallee string

func (g getPropCallee) exec(vm *vm) {
	v := vm.stack[vm.sp-1]
	obj := v.baseObject(vm.r)
	if obj == nil {
		vm.r.typeErrorResult(true, "Cannot read property '%s' of undefined", g)
	}
	prop := obj.self.getPropStr(string(g))
	if prop1, ok := prop.(*valueProperty); ok {
		vm.stack[vm.sp-1] = prop1.get(v)
	} else {
		if prop == nil {
			prop = memberUnresolved{valueUnresolved{r: vm.r, ref: string(g)}}
		}
		vm.stack[vm.sp-1] = prop
	}

	vm.pc++
}

type _getElem struct{}

var getElem _getElem

func (_getElem) exec(vm *vm) {
	v := vm.stack[vm.sp-2]
	obj := v.baseObject(vm.r)
	propName := vm.stack[vm.sp-1]
	if obj == nil {
		vm.r.typeErrorResult(true, "Cannot read property '%s' of undefined", propName.String())
	}

	prop := obj.self.getProp(propName)
	if prop1, ok := prop.(*valueProperty); ok {
		vm.stack[vm.sp-2] = prop1.get(v)
	} else {
		if prop == nil {
			prop = _undefined
		}
		vm.stack[vm.sp-2] = prop
	}

	vm.sp--
	vm.pc++
}

type _getElemCallee struct{}

var getElemCallee _getElemCallee

func (_getElemCallee) exec(vm *vm) {
	v := vm.stack[vm.sp-2]
	obj := v.baseObject(vm.r)
	propName := vm.stack[vm.sp-1]
	if obj == nil {
		vm.r.typeErrorResult(true, "Cannot read property '%s' of undefined", propName.String())
		panic("Unreachable")
	}

	prop := obj.self.getProp(propName)
	if prop1, ok := prop.(*valueProperty); ok {
		vm.stack[vm.sp-2] = prop1.get(v)
	} else {
		if prop == nil {
			prop = memberUnresolved{valueUnresolved{r: vm.r, ref: propName.String()}}
		}
		vm.stack[vm.sp-2] = prop
	}

	vm.sp--
	vm.pc++
}

type _dup struct{}

var dup _dup

func (_dup) exec(vm *vm) {
	vm.push(vm.stack[vm.sp-1])
	vm.pc++
}

type dupN uint32

func (d dupN) exec(vm *vm) {
	vm.push(vm.stack[vm.sp-1-int(d)])
	vm.pc++
}

type rdupN uint32

func (d rdupN) exec(vm *vm) {
	vm.stack[vm.sp-1-int(d)] = vm.stack[vm.sp-1]
	vm.pc++
}

type _newObject struct{}

var newObject _newObject

func (_newObject) exec(vm *vm) {
	vm.push(vm.r.NewObject())
	vm.pc++
}

type newArray uint32

func (l newArray) exec(vm *vm) {
	values := make([]Value, l)
	if l > 0 {
		copy(values, vm.stack[vm.sp-int(l):vm.sp])
	}
	obj := vm.r.newArrayValues(values)
	if l > 0 {
		vm.sp -= int(l) - 1
		vm.stack[vm.sp-1] = obj
	} else {
		vm.push(obj)
	}
	vm.pc++
}

type newRegexp struct {
	pattern regexpPattern
	src     valueString

	global, ignoreCase, multiline bool
}

func (n *newRegexp) exec(vm *vm) {
	vm.push(vm.r.newRegExpp(n.pattern, n.src, n.global, n.ignoreCase, n.multiline, vm.r.global.RegExpPrototype))
	vm.pc++
}

func (vm *vm) setLocal(s int) {
	v := vm.stack[vm.sp-1]
	level := s >> 24
	idx := uint32(s & 0x00FFFFFF)
	stash := vm.stash
	for i := 0; i < level; i++ {
		stash = stash.outer
	}
	stash.putByIdx(idx, v)
	vm.pc++
}

type setLocal uint32

func (s setLocal) exec(vm *vm) {
	vm.setLocal(int(s))
}

type setLocalP uint32

func (s setLocalP) exec(vm *vm) {
	vm.setLocal(int(s))
	vm.sp--
}

type setVar struct {
	name string
	idx  uint32
}

func (s setVar) exec(vm *vm) {
	v := vm.peek()

	level := int(s.idx >> 24)
	idx := uint32(s.idx & 0x00FFFFFF)
	stash := vm.stash
	name := s.name
	for i := 0; i < level; i++ {
		if stash.put(name, v) {
			goto end
		}
		stash = stash.outer
	}

	if stash != nil {
		stash.putByIdx(idx, v)
	} else {
		vm.r.globalObject.self.putStr(name, v, false)
	}

end:
	vm.pc++
}

type resolveVar1 string

func (s resolveVar1) exec(vm *vm) {
	name := string(s)
	var ref ref
	for stash := vm.stash; stash != nil; stash = stash.outer {
		if stash.obj != nil {
			if stash.obj.hasPropertyStr(name) {
				ref = &objRef{
					base: stash.obj,
					name: name,
				}
				goto end
			}
		} else {
			if idx, exists := stash.names[name]; exists {
				ref = &stashRef{
					v: &stash.values[idx],
				}
				goto end
			}
		}
	}

	ref = &objRef{
		base: vm.r.globalObject.self,
		name: name,
	}

end:
	vm.refStack = append(vm.refStack, ref)
	vm.pc++
}

type deleteVar string

func (d deleteVar) exec(vm *vm) {
	name := string(d)
	ret := true
	for stash := vm.stash; stash != nil; stash = stash.outer {
		if stash.obj != nil {
			if stash.obj.hasPropertyStr(name) {
				ret = stash.obj.deleteStr(name, false)
				goto end
			}
		} else {
			if _, exists := stash.names[name]; exists {
				ret = false
				goto end
			}
		}
	}

	if vm.r.globalObject.self.hasPropertyStr(name) {
		ret = vm.r.globalObject.self.deleteStr(name, false)
	}

end:
	if ret {
		vm.push(valueTrue)
	} else {
		vm.push(valueFalse)
	}
	vm.pc++
}

type deleteGlobal string

func (d deleteGlobal) exec(vm *vm) {
	name := string(d)
	var ret bool
	if vm.r.globalObject.self.hasPropertyStr(name) {
		ret = vm.r.globalObject.self.deleteStr(name, false)
	} else {
		ret = true
	}
	if ret {
		vm.push(valueTrue)
	} else {
		vm.push(valueFalse)
	}
	vm.pc++
}

type resolveVar1Strict string

func (s resolveVar1Strict) exec(vm *vm) {
	name := string(s)
	var ref ref
	for stash := vm.stash; stash != nil; stash = stash.outer {
		if stash.obj != nil {
			if stash.obj.hasPropertyStr(name) {
				ref = &objRef{
					base:   stash.obj,
					name:   name,
					strict: true,
				}
				goto end
			}
		} else {
			if idx, exists := stash.names[name]; exists {
				ref = &stashRef{
					v: &stash.values[idx],
				}
				goto end
			}
		}
	}

	if vm.r.globalObject.self.hasPropertyStr(name) {
		ref = &objRef{
			base:   vm.r.globalObject.self,
			name:   name,
			strict: true,
		}
		goto end
	}

	ref = &unresolvedRef{
		runtime: vm.r,
		name:    string(s),
	}

end:
	vm.refStack = append(vm.refStack, ref)
	vm.pc++
}

type setGlobal string

func (s setGlobal) exec(vm *vm) {
	v := vm.peek()

	vm.r.globalObject.self.putStr(string(s), v, false)
	vm.pc++
}

type setVarStrict struct {
	name string
	idx  uint32
}

func (s setVarStrict) exec(vm *vm) {
	v := vm.peek()

	level := int(s.idx >> 24)
	idx := uint32(s.idx & 0x00FFFFFF)
	stash := vm.stash
	name := s.name
	for i := 0; i < level; i++ {
		if stash.put(name, v) {
			goto end
		}
		stash = stash.outer
	}

	if stash != nil {
		stash.putByIdx(idx, v)
	} else {
		o := vm.r.globalObject.self
		if o.hasOwnPropertyStr(name) {
			o.putStr(name, v, true)
		} else {
			vm.r.throwReferenceError(name)
		}
	}

end:
	vm.pc++
}

type setVar1Strict string

func (s setVar1Strict) exec(vm *vm) {
	v := vm.peek()
	var o objectImpl

	name := string(s)
	for stash := vm.stash; stash != nil; stash = stash.outer {
		if stash.put(name, v) {
			goto end
		}
	}
	o = vm.r.globalObject.self
	if o.hasOwnPropertyStr(name) {
		o.putStr(name, v, true)
	} else {
		vm.r.throwReferenceError(name)
	}
end:
	vm.pc++
}

type setGlobalStrict string

func (s setGlobalStrict) exec(vm *vm) {
	v := vm.peek()

	name := string(s)
	o := vm.r.globalObject.self
	if o.hasOwnPropertyStr(name) {
		o.putStr(name, v, true)
	} else {
		vm.r.throwReferenceError(name)
	}
	vm.pc++
}

type getLocal uint32

func (g getLocal) exec(vm *vm) {
	level := int(g >> 24)
	idx := uint32(g & 0x00FFFFFF)
	stash := vm.stash
	for i := 0; i < level; i++ {
		stash = stash.outer
	}

	vm.push(stash.getByIdx(idx))
	vm.pc++
}

type getVar struct {
	name string
	idx  uint32
	ref  bool
}

func (g getVar) exec(vm *vm) {
	level := int(g.idx >> 24)
	idx := uint32(g.idx & 0x00FFFFFF)
	stash := vm.stash
	name := g.name
	for i := 0; i < level; i++ {
		if v, found := stash.getByName(name, vm); found {
			vm.push(v)
			goto end
		}
		stash = stash.outer
	}
	if stash != nil {
		vm.push(stash.getByIdx(idx))
	} else {
		v := vm.r.globalObject.self.getStr(name)
		if v == nil {
			if g.ref {
				v = valueUnresolved{r: vm.r, ref: name}
			} else {
				vm.r.throwReferenceError(name)
			}
		}
		vm.push(v)
	}
end:
	vm.pc++
}

type resolveVar struct {
	name   string
	idx    uint32
	strict bool
}

func (r resolveVar) exec(vm *vm) {
	level := int(r.idx >> 24)
	idx := uint32(r.idx & 0x00FFFFFF)
	stash := vm.stash
	var ref ref
	for i := 0; i < level; i++ {
		if stash.obj != nil {
			if stash.obj.hasPropertyStr(r.name) {
				ref = &objRef{
					base:   stash.obj,
					name:   r.name,
					strict: r.strict,
				}
				goto end
			}
		} else {
			if idx, exists := stash.names[r.name]; exists {
				ref = &stashRef{
					v: &stash.values[idx],
				}
				goto end
			}
		}
		stash = stash.outer
	}

	if stash != nil {
		ref = &stashRef{
			v: &stash.values[idx],
		}
		goto end
	} /*else {
		if vm.r.globalObject.self.hasProperty(nameVal) {
			ref = &objRef{
				base: vm.r.globalObject.self,
				name: r.name,
			}
			goto end
		}
	} */

	ref = &unresolvedRef{
		runtime: vm.r,
		name:    r.name,
	}

end:
	vm.refStack = append(vm.refStack, ref)
	vm.pc++
}

type _getValue struct{}

var getValue _getValue

func (_getValue) exec(vm *vm) {
	ref := vm.refStack[len(vm.refStack)-1]
	if v := ref.get(); v != nil {
		vm.push(v)
	} else {
		vm.r.throwReferenceError(ref.refname())
		panic("Unreachable")
	}
	vm.pc++
}

type _putValue struct{}

var putValue _putValue

func (_putValue) exec(vm *vm) {
	l := len(vm.refStack) - 1
	ref := vm.refStack[l]
	vm.refStack[l] = nil
	vm.refStack = vm.refStack[:l]
	ref.set(vm.stack[vm.sp-1])
	vm.pc++
}

type getVar1 string

func (n getVar1) exec(vm *vm) {
	name := string(n)
	var val Value
	for stash := vm.stash; stash != nil; stash = stash.outer {
		if v, exists := stash.getByName(name, vm); exists {
			val = v
			break
		}
	}
	if val == nil {
		val = vm.r.globalObject.self.getStr(name)
		if val == nil {
			vm.r.throwReferenceError(name)
		}
	}
	vm.push(val)
	vm.pc++
}

type getVar1Callee string

func (n getVar1Callee) exec(vm *vm) {
	name := string(n)
	var val Value
	for stash := vm.stash; stash != nil; stash = stash.outer {
		if v, exists := stash.getByName(name, vm); exists {
			val = v
			break
		}
	}
	if val == nil {
		val = vm.r.globalObject.self.getStr(name)
		if val == nil {
			val = valueUnresolved{r: vm.r, ref: name}
		}
	}
	vm.push(val)
	vm.pc++
}

type _pop struct{}

var pop _pop

func (_pop) exec(vm *vm) {
	vm.sp--
	vm.pc++
}

type _swap struct{}

var swap _swap

func (_swap) exec(vm *vm) {
	vm.stack[vm.sp-1], vm.stack[vm.sp-2] = vm.stack[vm.sp-2], vm.stack[vm.sp-1]
	vm.pc++
}

func (vm *vm) callEval(n int, strict bool) {
	if vm.r.toObject(vm.stack[vm.sp-n-1]) == vm.r.global.Eval {
		if n > 0 {
			srcVal := vm.stack[vm.sp-n]
			if src, ok := srcVal.assertString(); ok {
				var this Value
				if vm.sb != 0 {
					this = vm.stack[vm.sb]
				} else {
					this = vm.r.globalObject
				}
				ret := vm.r.eval(src.String(), true, strict, this)
				vm.stack[vm.sp-n-2] = ret
			} else {
				vm.stack[vm.sp-n-2] = srcVal
			}
		} else {
			vm.stack[vm.sp-n-2] = _undefined
		}

		vm.sp -= n + 1
		vm.pc++
	} else {
		call(n).exec(vm)
	}
}

type callEval uint32

func (numargs callEval) exec(vm *vm) {
	vm.callEval(int(numargs), false)
}

type callEvalStrict uint32

func (numargs callEvalStrict) exec(vm *vm) {
	vm.callEval(int(numargs), true)
}

type _boxThis struct{}

var boxThis _boxThis

func (_boxThis) exec(vm *vm) {
	v := vm.stack[vm.sb]
	if v == _undefined || v == _null {
		vm.stack[vm.sb] = vm.r.globalObject
	} else {
		vm.stack[vm.sb] = v.ToObject(vm.r)
	}
	vm.pc++
}

type call uint32

func (numargs call) exec(vm *vm) {
	// this
	// callee
	// arg0
	// ...
	// arg<numargs-1>
	n := int(numargs)
	v := vm.stack[vm.sp-n-1] // callee
	obj := vm.r.toCallee(v)
repeat:
	switch f := obj.self.(type) {
	case *funcObject:
		vm.pc++
		vm.pushCtx()
		vm.args = n
		vm.prg = f.prg
		vm.stash = f.stash
		vm.pc = 0
		vm.stack[vm.sp-n-1], vm.stack[vm.sp-n-2] = vm.stack[vm.sp-n-2], vm.stack[vm.sp-n-1]
		return
	case *nativeFuncObject:
		vm._nativeCall(f, n)
	case *boundFuncObject:
		vm._nativeCall(&f.nativeFuncObject, n)
	case *lazyObject:
		obj.self = f.create(obj)
		goto repeat
	default:
		vm.r.typeErrorResult(true, "Not a function: %s", obj.ToString())
	}
}

func (vm *vm) _nativeCall(f *nativeFuncObject, n int) {
	if f.f != nil {
		vm.pushCtx()
		vm.prg = nil
		vm.funcName = f.nameProp.get(nil).String()
		ret := f.f(FunctionCall{
			Arguments: vm.stack[vm.sp-n : vm.sp],
			This:      vm.stack[vm.sp-n-2],
		})
		if ret == nil {
			ret = _undefined
		}
		vm.stack[vm.sp-n-2] = ret
		vm.popCtx()
	} else {
		vm.stack[vm.sp-n-2] = _undefined
	}
	vm.sp -= n + 1
	vm.pc++
}

func (vm *vm) clearStack() {
	stackTail := vm.stack[vm.sp:]
	for i := range stackTail {
		stackTail[i] = nil
	}
	vm.stack = vm.stack[:vm.sp]
}

type enterFunc uint32

func (e enterFunc) exec(vm *vm) {
	// Input stack:
	//
	// callee
	// this
	// arg0
	// ...
	// argN
	// <- sp

	// Output stack:
	//
	// this <- sb
	// <- sp

	vm.newStash()
	offset := vm.args - int(e)
	vm.stash.values = make([]Value, e)
	if offset > 0 {
		copy(vm.stash.values, vm.stack[vm.sp-vm.args:])
		vm.stash.extraArgs = make([]Value, offset)
		copy(vm.stash.extraArgs, vm.stack[vm.sp-offset:])
	} else {
		copy(vm.stash.values, vm.stack[vm.sp-vm.args:])
		vv := vm.stash.values[vm.args:]
		for i, _ := range vv {
			vv[i] = _undefined
		}
	}
	vm.sp -= vm.args
	vm.sb = vm.sp - 1
	vm.pc++
}

type _ret struct{}

var ret _ret

func (_ret) exec(vm *vm) {
	// callee -3
	// this -2
	// retval -1

	vm.stack[vm.sp-3] = vm.stack[vm.sp-1]
	vm.sp -= 2
	vm.popCtx()
	if vm.pc < 0 {
		vm.halt = true
	}
}

type enterFuncStashless struct {
	stackSize uint32
	args      uint32
}

func (e enterFuncStashless) exec(vm *vm) {
	vm.sb = vm.sp - vm.args - 1
	var ss int
	d := int(e.args) - vm.args
	if d > 0 {
		ss = int(e.stackSize) + d
		vm.args = int(e.args)
	} else {
		ss = int(e.stackSize)
	}
	sp := vm.sp
	if ss > 0 {
		vm.sp += int(ss)
		vm.stack.expand(vm.sp)
		s := vm.stack[sp:vm.sp]
		for i, _ := range s {
			s[i] = _undefined
		}
	}
	vm.pc++
}

type _retStashless struct{}

var retStashless _retStashless

func (_retStashless) exec(vm *vm) {
	retval := vm.stack[vm.sp-1]
	vm.sp = vm.sb
	vm.stack[vm.sp-1] = retval
	vm.popCtx()
	if vm.pc < 0 {
		vm.halt = true
	}
}

type newFunc struct {
	prg    *Program
	name   string
	length uint32
	strict bool

	srcStart, srcEnd uint32
}

func (n *newFunc) exec(vm *vm) {
	obj := vm.r.newFunc(n.name, int(n.length), n.strict)
	obj.prg = n.prg
	obj.stash = vm.stash
	obj.src = n.prg.src.src[n.srcStart:n.srcEnd]
	vm.push(obj.val)
	vm.pc++
}

type bindName string

func (d bindName) exec(vm *vm) {
	if vm.stash != nil {
		vm.stash.createBinding(string(d))
	} else {
		vm.r.globalObject.self._putProp(string(d), _undefined, true, true, false)
	}
	vm.pc++
}

type jne int32

func (j jne) exec(vm *vm) {
	vm.sp--
	if !vm.stack[vm.sp].ToBoolean() {
		vm.pc += int(j)
	} else {
		vm.pc++
	}
}

type jeq int32

func (j jeq) exec(vm *vm) {
	vm.sp--
	if vm.stack[vm.sp].ToBoolean() {
		vm.pc += int(j)
	} else {
		vm.pc++
	}
}

type jeq1 int32

func (j jeq1) exec(vm *vm) {
	if vm.stack[vm.sp-1].ToBoolean() {
		vm.pc += int(j)
	} else {
		vm.pc++
	}
}

type jneq1 int32

func (j jneq1) exec(vm *vm) {
	if !vm.stack[vm.sp-1].ToBoolean() {
		vm.pc += int(j)
	} else {
		vm.pc++
	}
}

type _not struct{}

var not _not

func (_not) exec(vm *vm) {
	if vm.stack[vm.sp-1].ToBoolean() {
		vm.stack[vm.sp-1] = valueFalse
	} else {
		vm.stack[vm.sp-1] = valueTrue
	}
	vm.pc++
}

func toPrimitiveNumber(v Value) Value {
	if o, ok := v.(*Object); ok {
		return o.self.toPrimitiveNumber()
	}
	return v
}

func cmp(px, py Value) Value {
	var ret bool
	var nx, ny float64

	if xs, ok := px.assertString(); ok {
		if ys, ok := py.assertString(); ok {
			ret = xs.compareTo(ys) < 0
			goto end
		}
	}

	if xi, ok := px.assertInt(); ok {
		if yi, ok := py.assertInt(); ok {
			ret = xi < yi
			goto end
		}
	}

	nx = px.ToFloat()
	ny = py.ToFloat()

	if math.IsNaN(nx) || math.IsNaN(ny) {
		return _undefined
	}

	ret = nx < ny

end:
	if ret {
		return valueTrue
	}
	return valueFalse

}

type _op_lt struct{}

var op_lt _op_lt

func (_op_lt) exec(vm *vm) {
	left := toPrimitiveNumber(vm.stack[vm.sp-2])
	right := toPrimitiveNumber(vm.stack[vm.sp-1])

	r := cmp(left, right)
	if r == _undefined {
		vm.stack[vm.sp-2] = valueFalse
	} else {
		vm.stack[vm.sp-2] = r
	}
	vm.sp--
	vm.pc++
}

type _op_lte struct{}

var op_lte _op_lte

func (_op_lte) exec(vm *vm) {
	left := toPrimitiveNumber(vm.stack[vm.sp-2])
	right := toPrimitiveNumber(vm.stack[vm.sp-1])

	r := cmp(right, left)
	if r == _undefined || r == valueTrue {
		vm.stack[vm.sp-2] = valueFalse
	} else {
		vm.stack[vm.sp-2] = valueTrue
	}

	vm.sp--
	vm.pc++
}

type _op_gt struct{}

var op_gt _op_gt

func (_op_gt) exec(vm *vm) {
	left := toPrimitiveNumber(vm.stack[vm.sp-2])
	right := toPrimitiveNumber(vm.stack[vm.sp-1])

	r := cmp(right, left)
	if r == _undefined {
		vm.stack[vm.sp-2] = valueFalse
	} else {
		vm.stack[vm.sp-2] = r
	}
	vm.sp--
	vm.pc++
}

type _op_gte struct{}

var op_gte _op_gte

func (_op_gte) exec(vm *vm) {
	left := toPrimitiveNumber(vm.stack[vm.sp-2])
	right := toPrimitiveNumber(vm.stack[vm.sp-1])

	r := cmp(left, right)
	if r == _undefined || r == valueTrue {
		vm.stack[vm.sp-2] = valueFalse
	} else {
		vm.stack[vm.sp-2] = valueTrue
	}

	vm.sp--
	vm.pc++
}

type _op_eq struct{}

var op_eq _op_eq

func (_op_eq) exec(vm *vm) {
	if vm.stack[vm.sp-2].Equals(vm.stack[vm.sp-1]) {
		vm.stack[vm.sp-2] = valueTrue
	} else {
		vm.stack[vm.sp-2] = valueFalse
	}
	vm.sp--
	vm.pc++
}

type _op_neq struct{}

var op_neq _op_neq

func (_op_neq) exec(vm *vm) {
	if vm.stack[vm.sp-2].Equals(vm.stack[vm.sp-1]) {
		vm.stack[vm.sp-2] = valueFalse
	} else {
		vm.stack[vm.sp-2] = valueTrue
	}
	vm.sp--
	vm.pc++
}

type _op_strict_eq struct{}

var op_strict_eq _op_strict_eq

func (_op_strict_eq) exec(vm *vm) {
	if vm.stack[vm.sp-2].StrictEquals(vm.stack[vm.sp-1]) {
		vm.stack[vm.sp-2] = valueTrue
	} else {
		vm.stack[vm.sp-2] = valueFalse
	}
	vm.sp--
	vm.pc++
}

type _op_strict_neq struct{}

var op_strict_neq _op_strict_neq

func (_op_strict_neq) exec(vm *vm) {
	if vm.stack[vm.sp-2].StrictEquals(vm.stack[vm.sp-1]) {
		vm.stack[vm.sp-2] = valueFalse
	} else {
		vm.stack[vm.sp-2] = valueTrue
	}
	vm.sp--
	vm.pc++
}

type _op_instanceof struct{}

var op_instanceof _op_instanceof

func (_op_instanceof) exec(vm *vm) {
	left := vm.stack[vm.sp-2]
	right := vm.r.toObject(vm.stack[vm.sp-1])

	if right.self.hasInstance(left) {
		vm.stack[vm.sp-2] = valueTrue
	} else {
		vm.stack[vm.sp-2] = valueFalse
	}

	vm.sp--
	vm.pc++
}

type _op_in struct{}

var op_in _op_in

func (_op_in) exec(vm *vm) {
	left := vm.stack[vm.sp-2]
	right := vm.r.toObject(vm.stack[vm.sp-1])

	if right.self.hasProperty(left) {
		vm.stack[vm.sp-2] = valueTrue
	} else {
		vm.stack[vm.sp-2] = valueFalse
	}

	vm.sp--
	vm.pc++
}

type try struct {
	catchOffset   int32
	finallyOffset int32
	dynamic       bool
}

func (t try) exec(vm *vm) {
	o := vm.pc
	vm.pc++
	ex := vm.runTry()
	if ex != nil && t.catchOffset > 0 {
		// run the catch block (in try)
		vm.pc = o + int(t.catchOffset)
		// TODO: if ex.val is an Error, set the stack property
		if t.dynamic {
			vm.newStash()
			vm.stash.putByIdx(0, ex.val)
		} else {
			vm.push(ex.val)
		}
		ex = vm.runTry()
		if t.dynamic {
			vm.stash = vm.stash.outer
		}
	}

	if t.finallyOffset > 0 {
		pc := vm.pc
		// Run finally
		vm.pc = o + int(t.finallyOffset)
		vm.run()
		if vm.prg.code[vm.pc] == retFinally {
			vm.pc = pc
		} else {
			// break or continue out of finally, dropping exception
			ex = nil
		}
	}

	vm.halt = false

	if ex != nil {
		panic(ex)
	}
}

type _retFinally struct{}

var retFinally _retFinally

func (_retFinally) exec(vm *vm) {
	vm.pc++
}

type enterCatch string

func (varName enterCatch) exec(vm *vm) {
	vm.stash.names = map[string]uint32{
		string(varName): 0,
	}
	vm.pc++
}

type _throw struct{}

var throw _throw

func (_throw) exec(vm *vm) {
	panic(vm.stack[vm.sp-1])
}

type _new uint32

func (n _new) exec(vm *vm) {
	obj := vm.r.toObject(vm.stack[vm.sp-1-int(n)])
repeat:
	switch f := obj.self.(type) {
	case *funcObject:
		args := make([]Value, n)
		copy(args, vm.stack[vm.sp-int(n):])
		vm.sp -= int(n)
		vm.stack[vm.sp-1] = f.construct(args)
	case *nativeFuncObject:
		vm._nativeNew(f, int(n))
	case *boundFuncObject:
		vm._nativeNew(&f.nativeFuncObject, int(n))
	case *lazyObject:
		obj.self = f.create(obj)
		goto repeat
	default:
		vm.r.typeErrorResult(true, "Not a constructor")
	}

	vm.pc++
}

func (vm *vm) _nativeNew(f *nativeFuncObject, n int) {
	if f.construct != nil {
		args := make([]Value, n)
		copy(args, vm.stack[vm.sp-n:])
		vm.sp -= n
		vm.stack[vm.sp-1] = f.construct(args)
	} else {
		vm.r.typeErrorResult(true, "Not a constructor")
	}
}

type _typeof struct{}

var typeof _typeof

func (_typeof) exec(vm *vm) {
	var r Value
	switch v := vm.stack[vm.sp-1].(type) {
	case valueUndefined, valueUnresolved:
		r = stringUndefined
	case valueNull:
		r = stringObjectC
	case *Object:
	repeat:
		switch s := v.self.(type) {
		case *funcObject, *nativeFuncObject, *boundFuncObject:
			r = stringFunction
		case *lazyObject:
			v.self = s.create(v)
			goto repeat
		default:
			r = stringObjectC
		}
	case valueBool:
		r = stringBoolean
	case valueString:
		r = stringString
	case valueInt, valueFloat:
		r = stringNumber
	default:
		panic(fmt.Errorf("Unknown type: %T", v))
	}
	vm.stack[vm.sp-1] = r
	vm.pc++
}

type createArgs uint32

func (formalArgs createArgs) exec(vm *vm) {
	v := &Object{runtime: vm.r}
	args := &argumentsObject{}
	args.extensible = true
	args.prototype = vm.r.global.ObjectPrototype
	args.class = "Arguments"
	v.self = args
	args.val = v
	args.length = vm.args
	args.init()
	i := 0
	c := int(formalArgs)
	if vm.args < c {
		c = vm.args
	}
	for ; i < c; i++ {
		args._put(strconv.Itoa(i), &mappedProperty{
			valueProperty: valueProperty{
				writable:     true,
				configurable: true,
				enumerable:   true,
			},
			v: &vm.stash.values[i],
		})
	}

	for _, v := range vm.stash.extraArgs {
		args._put(strconv.Itoa(i), v)
		i++
	}

	args._putProp("callee", vm.stack[vm.sb-1], true, false, true)
	vm.push(v)
	vm.pc++
}

type createArgsStrict uint32

func (formalArgs createArgsStrict) exec(vm *vm) {
	args := vm.r.newBaseObject(vm.r.global.ObjectPrototype, "Arguments")
	i := 0
	c := int(formalArgs)
	if vm.args < c {
		c = vm.args
	}
	for _, v := range vm.stash.values[:c] {
		args._put(strconv.Itoa(i), v)
		i++
	}

	for _, v := range vm.stash.extraArgs {
		args._put(strconv.Itoa(i), v)
		i++
	}

	args._putProp("length", intToValue(int64(vm.args)), true, false, true)
	args._put("callee", vm.r.global.throwerProperty)
	args._put("caller", vm.r.global.throwerProperty)
	vm.push(args.val)
	vm.pc++
}

type _enterWith struct{}

var enterWith _enterWith

func (_enterWith) exec(vm *vm) {
	vm.newStash()
	vm.stash.obj = vm.stack[vm.sp-1].ToObject(vm.r).self
	vm.sp--
	vm.pc++
}

type _leaveWith struct{}

var leaveWith _leaveWith

func (_leaveWith) exec(vm *vm) {
	vm.stash = vm.stash.outer
	vm.pc++
}

func emptyIter() (propIterItem, iterNextFunc) {
	return propIterItem{}, nil
}

type _enumerate struct{}

var enumerate _enumerate

func (_enumerate) exec(vm *vm) {
	v := vm.stack[vm.sp-1]
	if v == _undefined || v == _null {
		vm.iterStack = append(vm.iterStack, iterStackItem{f: emptyIter})
	} else {
		vm.iterStack = append(vm.iterStack, iterStackItem{f: v.ToObject(vm.r).self.enumerate(false, true)})
	}
	vm.sp--
	vm.pc++
}

type enumNext int32

func (jmp enumNext) exec(vm *vm) {
	l := len(vm.iterStack) - 1
	item, n := vm.iterStack[l].f()
	if n != nil {
		vm.iterStack[l].val = newStringValue(item.name)
		vm.iterStack[l].f = n
		vm.pc++
	} else {
		vm.pc += int(jmp)
	}
}

type _enumGet struct{}

var enumGet _enumGet

func (_enumGet) exec(vm *vm) {
	l := len(vm.iterStack) - 1
	vm.push(vm.iterStack[l].val)
	vm.pc++
}

type _enumPop struct{}

var enumPop _enumPop

func (_enumPop) exec(vm *vm) {
	l := len(vm.iterStack) - 1
	vm.iterStack[l] = iterStackItem{}
	vm.iterStack = vm.iterStack[:l]
	vm.pc++
}
