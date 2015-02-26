package qml

// #cgo CPPFLAGS: -I./cpp
// #cgo CXXFLAGS: -std=c++0x -pedantic-errors -Wall -fno-strict-aliasing
// #cgo LDFLAGS: -lstdc++
// #cgo pkg-config: Qt5Core Qt5Widgets Qt5Quick
//
// #include <stdlib.h>
//
// #include "cpp/capi.h"
//
import "C"

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"

	"gopkg.in/qml.v1/cdata"
)

var (
	guiFunc      = make(chan func())
	guiDone      = make(chan struct{})
	guiLock      = 0
	guiMainRef   uintptr
	guiPaintRef  uintptr
	guiIdleRun   int32

	initialized int32
)

func init() {
	runtime.LockOSThread()
	guiMainRef = cdata.Ref()
}

// Run runs the main QML event loop, runs f, and then terminates the
// event loop once f returns.
//
// Most functions from the qml package block until Run is called.
//
// The Run function must necessarily be called from the same goroutine as
// the main function or the application may fail when running on Mac OS.
func Run(f func() error) error {
	if cdata.Ref() != guiMainRef {
		panic("Run must be called on the initial goroutine so apps are portable to Mac OS")
	}
	if !atomic.CompareAndSwapInt32(&initialized, 0, 1) {
		panic("qml.Run called more than once")
	}
	C.newGuiApplication()
	C.idleTimerInit((*C.int32_t)(&guiIdleRun))
	done := make(chan error, 1)
	go func() {
		RunMain(func() {}) // Block until the event loop is running.
		done <- f()
		C.applicationExit()
	}()
	C.applicationExec()
	return <-done
}

// RunMain runs f in the main QML thread and waits for f to return.
//
// This is meant to be used by extensions that integrate directly with the
// underlying QML logic.
func RunMain(f func()) {
	ref := cdata.Ref()
	if ref == guiMainRef || ref == atomic.LoadUintptr(&guiPaintRef) {
		// Already within the GUI or render threads. Attempting to wait would deadlock.
		f()
		return
	}

	// Tell Qt we're waiting for the idle hook to be called.
	if atomic.AddInt32(&guiIdleRun, 1) == 1 {
		C.idleTimerStart()
	}

	// Send f to be executed by the idle hook in the main GUI thread.
	guiFunc <- f

	// Wait until f is done executing.
	<-guiDone
}

// Lock freezes all QML activity by blocking the main event loop.
// Locking is necessary before updating shared data structures
// without race conditions.
//
// It's safe to use qml functionality while holding a lock, as
// long as the requests made do not depend on follow up QML
// events to be processed before returning. If that happens, the
// problem will be observed as the application freezing.
//
// The Lock function is reentrant. That means it may be called
// multiple times, and QML activities will only be resumed after
// Unlock is called a matching number of times.
func Lock() {
	// TODO Better testing for this.
	RunMain(func() {
		guiLock++
	})
}

// Unlock releases the QML event loop. See Lock for details.
func Unlock() {
	RunMain(func() {
		if guiLock == 0 {
			panic("qml.Unlock called without lock being held")
		}
		guiLock--
	})
}

// Flush synchronously flushes all pending QML activities.
func Flush() {
	// TODO Better testing for this.
	RunMain(func() {
		C.applicationFlushAll()
	})
}

// Changed notifies all QML bindings that the given field value has changed.
//
// For example:
//
//     qml.Changed(&value, &value.Field)
//
func Changed(value, fieldAddr interface{}) {
	valuev := reflect.ValueOf(value)
	fieldv := reflect.ValueOf(fieldAddr)
	for valuev.Kind() == reflect.Ptr {
		valuev = valuev.Elem()
	}
	if fieldv.Kind() != reflect.Ptr {
		panic("qml.Changed received non-address value as fieldAddr")
	}
	fieldv = fieldv.Elem()
	if fieldv.Type().Size() == 0 {
		panic("cannot report changes on zero-sized fields")
	}
	offset := fieldv.UnsafeAddr() - valuev.UnsafeAddr()
	if !(0 <= offset && offset < valuev.Type().Size()) {
		panic("provided field is not a member of the given value")
	}

	RunMain(func() {
		tinfo := typeInfo(value)
		for _, engine := range engines {
			fold := engine.values[value]
			for fold != nil {
				C.goValueActivate(fold.cvalue, tinfo, C.int(offset))
				fold = fold.next
			}
			// TODO typeNew might also be a linked list keyed by the gvalue.
			//      This would prevent the iteration and the deferrals.
			for fold, _ = range typeNew {
				if fold.gvalue == value {
					// Activate these later so they don't get recursively moved
					// out of typeNew while the iteration is still happening.
					defer C.goValueActivate(fold.cvalue, tinfo, C.int(offset))
				}
			}
		}
	})
}

// hookIdleTimer is run once per iteration of the Qt event loop,
// within the main GUI thread, but only if at least one goroutine
// has atomically incremented guiIdleRun.
//
//export hookIdleTimer
func hookIdleTimer() {
	var f func()
	for {
		select {
		case f = <-guiFunc:
		default:
			if guiLock > 0 {
				f = <-guiFunc
			} else {
				return
			}
		}
		f()
		guiDone <- struct{}{}
		atomic.AddInt32(&guiIdleRun, -1)
	}
}

type valueFold struct {
	engine *Engine
	gvalue interface{}
	cvalue unsafe.Pointer
	init   reflect.Value
	prev   *valueFold
	next   *valueFold
	owner  valueOwner
}

type valueOwner uint8

const (
	cppOwner = 1 << iota
	jsOwner
)

// wrapGoValue creates a new GoValue object in C++ land wrapping
// the Go value contained in the given interface.
//
// This must be run from the main GUI thread.
func wrapGoValue(engine *Engine, gvalue interface{}, owner valueOwner) (cvalue unsafe.Pointer) {
	gvaluev := reflect.ValueOf(gvalue)
	gvaluek := gvaluev.Kind()
	if gvaluek == reflect.Struct && !hashable(gvalue) {
		name := gvaluev.Type().Name()
		if name != "" {
			name = " (" + name + ")"
		}
		panic("cannot hand an unhashable struct value" + name + " to QML logic; use its address instead")
	}
	if gvaluek == reflect.Ptr && gvaluev.Elem().Kind() == reflect.Ptr {
		panic("cannot hand pointer of pointer to QML logic; use a simple pointer instead")
	}

	painting := cdata.Ref() == atomic.LoadUintptr(&guiPaintRef)

	// Cannot reuse a jsOwner because the QML runtime may choose to destroy
	// the value _after_ we hand it a new reference to the same value.
	// See issue #68 for details.
	prev, ok := engine.values[gvalue]
	if ok && (prev.owner == cppOwner || painting) {
		return prev.cvalue
	}

	if painting {
		panic("cannot allocate new objects while painting")
	}

	parent := nilPtr
	if owner == cppOwner {
		parent = engine.addr
	}
	fold := &valueFold{
		engine: engine,
		gvalue: gvalue,
		owner:  owner,
	}
	fold.cvalue = C.newGoValue(unsafe.Pointer(fold), typeInfo(gvalue), parent)
	if prev != nil {
		// Put new fold first so the single cppOwner, if any, is always the first entry.
		fold.next = prev
		prev.prev = fold
	}
	engine.values[gvalue] = fold

	//fmt.Printf("[DEBUG] value alive (wrapped): cvalue=%x gvalue=%x/%#v\n", fold.cvalue, addrOf(fold.gvalue), fold.gvalue)
	stats.valuesAlive(+1)
	C.engineSetContextForObject(engine.addr, fold.cvalue)
	switch owner {
	case cppOwner:
		C.engineSetOwnershipCPP(engine.addr, fold.cvalue)
	case jsOwner:
		C.engineSetOwnershipJS(engine.addr, fold.cvalue)
	}
	return fold.cvalue
}

func addrOf(gvalue interface{}) uintptr {
	return reflect.ValueOf(gvalue).Pointer()
}

// typeNew holds fold values that are created by registered types.
// These values are special in two senses: first, they don't have a
// reference to an engine before they are used in a context that can
// set the reference; second, these values always hold a new cvalue,
// because they are created as a side-effect of the registered type
// being instantiated (it's too late to reuse an existent cvalue).
//
// For these reasons, typeNew holds the fold for these values until
// their engine is known, and once it's known they may have to be
// added to the linked list, since mulitple references for the same
// gvalue may occur.
var typeNew = make(map[*valueFold]bool)

//export hookGoValueTypeNew
func hookGoValueTypeNew(cvalue unsafe.Pointer, specp unsafe.Pointer) (foldp unsafe.Pointer) {
	// Initialization is postponed until the engine is available, so that
	// we can hand Init the qml.Object that represents the object.
	init := reflect.ValueOf((*TypeSpec)(specp).Init)
	fold := &valueFold{
		init:   init,
		gvalue: reflect.New(init.Type().In(0).Elem()).Interface(),
		cvalue: cvalue,
		owner:  jsOwner,
	}
	typeNew[fold] = true
	//fmt.Printf("[DEBUG] value alive (type-created): cvalue=%x gvalue=%x/%#v\n", fold.cvalue, addrOf(fold.gvalue), fold.gvalue)
	stats.valuesAlive(+1)
	return unsafe.Pointer(fold)
}

//export hookGoValueDestroyed
func hookGoValueDestroyed(enginep unsafe.Pointer, foldp unsafe.Pointer) {
	fold := (*valueFold)(foldp)
	engine := fold.engine
	if engine == nil {
		before := len(typeNew)
		delete(typeNew, fold)
		if len(typeNew) == before {
			panic("destroying value without an associated engine; who created the value?")
		}
	} else if engines[engine.addr] == nil {
		// Must never do that. The engine holds memory references that C++ depends on.
		panic(fmt.Sprintf("engine %p was released from global list while its values were still alive", engine.addr))
	} else {
		switch {
		case fold.prev != nil:
			fold.prev.next = fold.next
			if fold.next != nil {
				fold.next.prev = fold.prev
			}
		case fold.next != nil:
			fold.next.prev = fold.prev
			if fold.prev != nil {
				fold.prev.next = fold.next
			} else {
				fold.engine.values[fold.gvalue] = fold.next
			}
		default:
			before := len(engine.values)
			delete(engine.values, fold.gvalue)
			if len(engine.values) == before {
				panic("destroying value that knows about the engine, but the engine doesn't know about the value; who cleared the engine?")
			}
			if engine.destroyed && len(engine.values) == 0 {
				delete(engines, engine.addr)
			}
		}
	}
	//fmt.Printf("[DEBUG] value destroyed: cvalue=%x gvalue=%x/%#v\n", fold.cvalue, addrOf(fold.gvalue), fold.gvalue)
	stats.valuesAlive(-1)
}

func deref(value reflect.Value) reflect.Value {
	for {
		switch value.Kind() {
		case reflect.Ptr, reflect.Interface:
			value = value.Elem()
			continue
		}
		return value
	}
	panic("cannot happen")
}

//export hookGoValueReadField
func hookGoValueReadField(enginep, foldp unsafe.Pointer, reflectIndex, getIndex, setIndex C.int, resultdv *C.DataValue) {
	fold := ensureEngine(enginep, foldp)

	var field reflect.Value
	if getIndex >= 0 {
		field = reflect.ValueOf(fold.gvalue).Method(int(getIndex)).Call(nil)[0]
	} else {
		field = deref(reflect.ValueOf(fold.gvalue)).Field(int(reflectIndex))
	}
	field = deref(field)

	// Cannot compare Type directly as field may be invalid (nil).
	if field.Kind() == reflect.Slice && field.Type() == typeObjSlice {
		// TODO Handle getters that return []qml.Object.
		// TODO Handle other GoValue slices (!= []qml.Object).
		resultdv.dataType = C.DTListProperty
		*(*unsafe.Pointer)(unsafe.Pointer(&resultdv.data)) = C.newListProperty(foldp, C.intptr_t(reflectIndex), C.intptr_t(setIndex))
		return
	}

	fieldk := field.Kind()
	if fieldk == reflect.Slice || fieldk == reflect.Struct && field.Type() != typeRGBA {
		if field.CanAddr() {
			field = field.Addr()
		} else if !hashable(field.Interface()) {
			t := reflect.ValueOf(fold.gvalue).Type()
			for t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			panic(fmt.Sprintf("cannot access unaddressable and unhashable struct value on interface field %s.%s; value: %#v", t.Name(), t.Field(int(reflectIndex)).Name, field.Interface()))
		}
	}
	var gvalue interface{}
	if field.IsValid() {
		gvalue = field.Interface()
	}

	// TODO Strings are being passed in an unsafe manner here. There is a
	// small chance that the field is changed and the garbage collector is run
	// before C++ has a chance to look at the data. We can solve this problem
	// by queuing up values in a stack, and cleaning the stack when the
	// idle timer fires next.
	packDataValue(gvalue, resultdv, fold.engine, jsOwner)
}

//export hookGoValueWriteField
func hookGoValueWriteField(enginep, foldp unsafe.Pointer, reflectIndex, setIndex C.int, assigndv *C.DataValue) {
	fold := ensureEngine(enginep, foldp)
	v := reflect.ValueOf(fold.gvalue)
	ve := v
	for ve.Type().Kind() == reflect.Ptr {
		ve = ve.Elem()
	}
	var field, setMethod reflect.Value
	if reflectIndex >= 0 {
		// It's a real field rather than a getter.
		field = ve.Field(int(reflectIndex))
	}
	if setIndex >= 0 {
		// It has a setter.
		setMethod = v.Method(int(setIndex))
	}

	assign := unpackDataValue(assigndv, fold.engine)

	// TODO Return false to the call site if it fails. That's how Qt seems to handle it internally.
	err := convertAndSet(field, reflect.ValueOf(assign), setMethod)
	if err != nil {
		panic(err.Error())
	}
}

func convertAndSet(to, from reflect.Value, setMethod reflect.Value) (err error) {
	var toType reflect.Type
	if setMethod.IsValid() {
		toType = setMethod.Type().In(0)
	} else {
		toType = to.Type()
	}
	fromType := from.Type()
	defer func() {
		// TODO This is catching more than it should. There are calls
		//      to custom code below that should be isolated.
		if v := recover(); v != nil {
			err = fmt.Errorf("cannot use %s as a %s", fromType, toType)
		}
	}()
	if fromType == typeList && toType.Kind() == reflect.Slice {
		list := from.Interface().(*List)
		from = reflect.MakeSlice(toType, len(list.data), len(list.data))
		elemType := toType.Elem()
		for i, elem := range list.data {
			from.Index(i).Set(reflect.ValueOf(elem).Convert(elemType))
		}
	} else if fromType == typeMap && toType.Kind() == reflect.Map {
		qmap := from.Interface().(*Map)
		from = reflect.MakeMap(toType)
		elemType := toType.Elem()
		for i := 0; i < len(qmap.data); i += 2 {
			key := reflect.ValueOf(qmap.data[i])
			val := reflect.ValueOf(qmap.data[i+1])
			if val.Type() != elemType {
				val = val.Convert(elemType)
			}
			from.SetMapIndex(key, val)
		}
	} else if toType != fromType {
		from = from.Convert(toType)
	}
	if setMethod.IsValid() {
		setMethod.Call([]reflect.Value{from})
	} else {
		to.Set(from)
	}
	return nil
}

var (
	dataValueSize  = uintptr(unsafe.Sizeof(C.DataValue{}))
	dataValueArray [C.MaxParams]C.DataValue
)

//export hookGoValueCallMethod
func hookGoValueCallMethod(enginep, foldp unsafe.Pointer, reflectIndex C.int, args *C.DataValue) {
	fold := ensureEngine(enginep, foldp)
	v := reflect.ValueOf(fold.gvalue)

	// TODO Must assert that v is necessarily a pointer here, but we shouldn't have to manipulate
	//      gvalue here for that. This should happen in a sensible place in the wrapping functions
	//      that can still error out to the user in due time.

	method := v.Method(int(reflectIndex))
	methodt := method.Type()
	methodName := v.Type().Method(int(reflectIndex)).Name

	// TODO Ensure methods with more parameters than this are not registered.
	var params [C.MaxParams]reflect.Value
	var err error

	numIn := methodt.NumIn()
	for i := 0; i < numIn; i++ {
		paramdv := (*C.DataValue)(unsafe.Pointer(uintptr(unsafe.Pointer(args)) + (uintptr(i)+1)*dataValueSize))
		param := reflect.ValueOf(unpackDataValue(paramdv, fold.engine))
		if argt := methodt.In(i); param.Type() != argt {
			param, err = convertParam(methodName, i, param, argt)
			if err != nil {
				panic(err.Error())
			}
		}
		params[i] = param
	}

	result := method.Call(params[:numIn])

	if len(result) == 1 {
		packDataValue(result[0].Interface(), args, fold.engine, jsOwner)
	} else if len(result) > 1 {
		if len(result) > len(dataValueArray) {
			panic("function has too many results")
		}
		for i, v := range result {
			packDataValue(v.Interface(), &dataValueArray[i], fold.engine, jsOwner)
		}
		args.dataType = C.DTVariantList
		*(*unsafe.Pointer)(unsafe.Pointer(&args.data)) = C.newVariantList(&dataValueArray[0], C.int(len(result)))
	}
}

func convertParam(methodName string, index int, param reflect.Value, argt reflect.Type) (reflect.Value, error) {
	out := reflect.New(argt).Elem()
	err := convertAndSet(out, param, reflect.Value{})
	if err != nil {
		err = fmt.Errorf("cannot convert parameter %d of method %s from %s to %s; provided value: %#v",
			index, methodName, param.Type(), argt, param.Interface())
		return reflect.Value{}, err
	}
	return out, nil
}

func printPaintPanic() {
	if v := recover(); v != nil {
		buf := make([]byte, 8192)
		runtime.Stack(buf, false)
		fmt.Fprintf(os.Stderr, "panic while painting: %s\n\n%s", v, buf)
	}
}

//export hookGoValuePaint
func hookGoValuePaint(enginep, foldp unsafe.Pointer, reflectIndex C.intptr_t) {
	// Besides a convenience this is a workaround for http://golang.org/issue/8588
	defer printPaintPanic()
	defer atomic.StoreUintptr(&guiPaintRef, 0)

	// The main GUI thread is mutex-locked while paint methods are called,
	// so no two paintings should be happening at the same time.
	atomic.StoreUintptr(&guiPaintRef, cdata.Ref())

	fold := ensureEngine(enginep, foldp)
	if fold.init.IsValid() {
		return
	}

	painter := &Painter{engine: fold.engine, obj: &Common{fold.cvalue, fold.engine}}
	v := reflect.ValueOf(fold.gvalue)
	method := v.Method(int(reflectIndex))
	method.Call([]reflect.Value{reflect.ValueOf(painter)})
}

func ensureEngine(enginep, foldp unsafe.Pointer) *valueFold {
	fold := (*valueFold)(foldp)
	if fold.engine != nil {
		if fold.init.IsValid() {
			initGoType(fold)
		}
		return fold
	}

	if enginep == nilPtr {
		panic("accessing value without an engine pointer; who created the value?")
	}
	engine := engines[enginep]
	if engine == nil {
		panic("unknown engine pointer; who created the engine?")
	}
	fold.engine = engine
	prev := engine.values[fold.gvalue]
	if prev != nil {
		for prev.next != nil {
			prev = prev.next
		}
		prev.next = fold
		fold.prev = prev
	} else {
		engine.values[fold.gvalue] = fold
	}
	before := len(typeNew)
	delete(typeNew, fold)
	if len(typeNew) == before {
		panic("value had no engine, but was not created by a registered type; who created the value?")
	}
	initGoType(fold)
	return fold
}

func initGoType(fold *valueFold) {
	if cdata.Ref() == atomic.LoadUintptr(&guiPaintRef) {
		go RunMain(func() { _initGoType(fold, true) })
	} else {
		_initGoType(fold, false)
	}
}

func _initGoType(fold *valueFold, schedulePaint bool) {
	if !fold.init.IsValid() {
		return
	}
	// TODO Would be good to preserve identity on the Go side. See unpackDataValue as well.
	obj := &Common{engine: fold.engine, addr: fold.cvalue}
	fold.init.Call([]reflect.Value{reflect.ValueOf(fold.gvalue), reflect.ValueOf(obj)})
	fold.init = reflect.Value{}
	if schedulePaint {
		obj.Call("update")
	}
}

//export hookPanic
func hookPanic(message *C.char) {
	defer C.free(unsafe.Pointer(message))
	panic(C.GoString(message))
}

func listSlice(fold *valueFold, reflectIndex C.intptr_t) *[]Object {
	field := deref(reflect.ValueOf(fold.gvalue)).Field(int(reflectIndex))
	return field.Addr().Interface().(*[]Object)
}

//export hookListPropertyAt
func hookListPropertyAt(foldp unsafe.Pointer, reflectIndex, setIndex C.intptr_t, index C.int) (objp unsafe.Pointer) {
	fold := (*valueFold)(foldp)
	slice := listSlice(fold, reflectIndex)
	return (*slice)[int(index)].Common().addr
}

//export hookListPropertyCount
func hookListPropertyCount(foldp unsafe.Pointer, reflectIndex, setIndex C.intptr_t) C.int {
	fold := (*valueFold)(foldp)
	slice := listSlice(fold, reflectIndex)
	return C.int(len(*slice))
}

//export hookListPropertyAppend
func hookListPropertyAppend(foldp unsafe.Pointer, reflectIndex, setIndex C.intptr_t, objp unsafe.Pointer) {
	fold := (*valueFold)(foldp)
	slice := listSlice(fold, reflectIndex)
	var objdv C.DataValue
	objdv.dataType = C.DTObject
	*(*unsafe.Pointer)(unsafe.Pointer(&objdv.data)) = objp
	newslice := append(*slice, unpackDataValue(&objdv, fold.engine).(Object))
	if setIndex >= 0 {
		reflect.ValueOf(fold.gvalue).Method(int(setIndex)).Call([]reflect.Value{reflect.ValueOf(newslice)})
	} else {
		*slice = newslice
	}
}

//export hookListPropertyClear
func hookListPropertyClear(foldp unsafe.Pointer, reflectIndex, setIndex C.intptr_t) {
	fold := (*valueFold)(foldp)
	slice := listSlice(fold, reflectIndex)
	newslice := (*slice)[0:0]
	if setIndex >= 0 {
		reflect.ValueOf(fold.gvalue).Method(int(setIndex)).Call([]reflect.Value{reflect.ValueOf(newslice)})
	} else {
		for i := range *slice {
			(*slice)[i] = nil
		}
		*slice = newslice
	}
}
