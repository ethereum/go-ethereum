package duktape

/*
#cgo !windows CFLAGS: -std=c99 -O3 -Wall -fomit-frame-pointer -fstrict-aliasing
#cgo windows CFLAGS: -O3 -Wall -fomit-frame-pointer -fstrict-aliasing
#cgo linux LDFLAGS: -lm
#cgo freebsd LDFLAGS: -lm
#cgo openbsd LDFLAGS: -lm

#include "duktape.h"
#include "duk_logging.h"
#include "duk_print_alert.h"
#include "duk_module_duktape.h"
#include "duk_console.h"
extern duk_ret_t goFunctionCall(duk_context *ctx);
extern void goFinalizeCall(duk_context *ctx);
*/
import "C"
import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"unsafe"
)

var reFuncName = regexp.MustCompile("^[a-z_][a-z0-9_]*([A-Z_][a-z0-9_]*)*$")

const (
	goFunctionPtrProp = "\xff" + "goFunctionPtrProp"
	goContextPtrProp  = "\xff" + "goContextPtrProp"
)

type Context struct {
	*context
}

// transmute replaces the value from Context with the value of pointer
func (c *Context) transmute(p unsafe.Pointer) {
	*c = *(*Context)(p)
}

// this is a pojo containing only the values of the Context
type context struct {
	sync.Mutex
	duk_context *C.duk_context
	fnIndex     *functionIndex
	timerIndex  *timerIndex
}

// New returns plain initialized duktape context object
// See: http://duktape.org/api.html#duk_create_heap_default
func New() *Context {
	d := &Context{
		&context{
			duk_context: C.duk_create_heap(nil, nil, nil, nil, nil),
			fnIndex:     newFunctionIndex(),
			timerIndex:  &timerIndex{},
		},
	}

	ctx := d.duk_context
	C.duk_logging_init(ctx, 0)
	C.duk_print_alert_init(ctx, 0)
	C.duk_module_duktape_init(ctx)
	C.duk_console_init(ctx, 0)

	return d
}

// Flags is a set of flags for controlling the behaviour of duktape.
type Flags struct {
	Logging    uint
	PrintAlert uint
	Console    uint
}

// FlagConsoleProxyWrapper is a Console flag.
// Use a proxy wrapper to make undefined methods (console.foo()) no-ops.
const FlagConsoleProxyWrapper = 1 << 0

// FlagConsoleFlush is a Console flag.
// Flush output after every call.
const FlagConsoleFlush = 1 << 1

// NewWithFlags returns plain initialized duktape context object
// You can control the behaviour of duktape by setting flags.
// See: http://duktape.org/api.html#duk_create_heap_default
func NewWithFlags(flags *Flags) *Context {
	d := &Context{
		&context{
			duk_context: C.duk_create_heap(nil, nil, nil, nil, nil),
			fnIndex:     newFunctionIndex(),
			timerIndex:  &timerIndex{},
		},
	}

	ctx := d.duk_context
	C.duk_logging_init(ctx, C.duk_uint_t(flags.Logging))
	C.duk_print_alert_init(ctx, C.duk_uint_t(flags.PrintAlert))
	C.duk_module_duktape_init(ctx)
	C.duk_console_init(ctx, C.duk_uint_t(flags.Console))

	return d
}

func contextFromPointer(ctx *C.duk_context) *Context {
	return &Context{&context{duk_context: ctx}}
}

// PushGlobalGoFunction push the given function into duktape global object
// Returns non-negative index (relative to stack bottom) of the pushed function
// also returns error if the function name is invalid
func (d *Context) PushGlobalGoFunction(name string, fn func(*Context) int) (int, error) {
	if !reFuncName.MatchString(name) {
		return -1, errors.New("Malformed function name '" + name + "'")
	}

	d.PushGlobalObject()
	idx := d.PushGoFunction(fn)
	d.PutPropString(-2, name)
	d.Pop()

	return idx, nil
}

// PushGoFunction push the given function into duktape stack, returns non-negative
// index (relative to stack bottom) of the pushed function
func (d *Context) PushGoFunction(fn func(*Context) int) int {
	funPtr := d.fnIndex.add(fn)
	ctxPtr := contexts.add(d)

	idx := d.PushCFunction((*[0]byte)(C.goFunctionCall), C.DUK_VARARGS)
	d.PushCFunction((*[0]byte)(C.goFinalizeCall), 1)
	d.PushPointer(funPtr)
	d.PutPropString(-2, goFunctionPtrProp)
	d.PushPointer(ctxPtr)
	d.PutPropString(-2, goContextPtrProp)
	d.SetFinalizer(-2)

	d.PushPointer(funPtr)
	d.PutPropString(-2, goFunctionPtrProp)
	d.PushPointer(ctxPtr)
	d.PutPropString(-2, goContextPtrProp)

	return idx
}

//export goFunctionCall
func goFunctionCall(cCtx *C.duk_context) C.duk_ret_t {
	d := contextFromPointer(cCtx)

	funPtr, ctx := d.getFunctionPtrs()
	d.transmute(unsafe.Pointer(ctx))

	result := d.fnIndex.get(funPtr)(d)

	return C.duk_ret_t(result)
}

//export goFinalizeCall
func goFinalizeCall(cCtx *C.duk_context) {
	d := contextFromPointer(cCtx)

	funPtr, ctx := d.getFunctionPtrs()
	d.transmute(unsafe.Pointer(ctx))

	d.fnIndex.delete(funPtr)
}

func (d *Context) getFunctionPtrs() (unsafe.Pointer, *Context) {
	d.PushCurrentFunction()
	d.GetPropString(-1, goFunctionPtrProp)
	funPtr := d.GetPointer(-1)

	d.Pop()

	d.GetPropString(-1, goContextPtrProp)
	ctx := contexts.get(d.GetPointer(-1))
	d.Pop2()
	return funPtr, ctx
}

// Destroy destroy all the references to the functions and freed the pointers
func (d *Context) Destroy() {
	d.fnIndex.destroy()
	contexts.delete(d)
}

type Error struct {
	Type       string
	Message    string
	FileName   string
	LineNumber int
	Stack      string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

type Type uint

func (t Type) IsNone() bool      { return t == TypeNone }
func (t Type) IsUndefined() bool { return t == TypeUndefined }
func (t Type) IsNull() bool      { return t == TypeNull }
func (t Type) IsBool() bool      { return t == TypeBoolean }
func (t Type) IsNumber() bool    { return t == TypeNumber }
func (t Type) IsString() bool    { return t == TypeString }
func (t Type) IsObject() bool    { return t == TypeObject }
func (t Type) IsBuffer() bool    { return t == TypeBuffer }
func (t Type) IsPointer() bool   { return t == TypePointer }
func (t Type) IsLightFunc() bool { return t == TypeLightFunc }

func (t Type) String() string {
	switch t {
	case TypeNone:
		return "None"
	case TypeUndefined:
		return "Undefined"
	case TypeNull:
		return "Null"
	case TypeBoolean:
		return "Boolean"
	case TypeNumber:
		return "Number"
	case TypeString:
		return "String"
	case TypeObject:
		return "Object"
	case TypeBuffer:
		return "Buffer"
	case TypePointer:
		return "Pointer"
	case TypeLightFunc:
		return "LightFunc"
	default:
		return "Unknown"
	}
}

type functionIndex struct {
	functions map[unsafe.Pointer]func(*Context) int
	sync.RWMutex
}

type timerIndex struct {
	c float64
	sync.Mutex
}

func (t *timerIndex) get() float64 {
	t.Lock()
	defer t.Unlock()
	t.c++
	return t.c
}

func newFunctionIndex() *functionIndex {
	return &functionIndex{
		functions: make(map[unsafe.Pointer]func(*Context) int, 0),
	}
}

func (i *functionIndex) add(fn func(*Context) int) unsafe.Pointer {
	ptr := C.malloc(1)

	i.Lock()
	i.functions[ptr] = fn
	i.Unlock()

	return ptr
}

func (i *functionIndex) get(ptr unsafe.Pointer) func(*Context) int {
	i.RLock()
	fn := i.functions[ptr]
	i.RUnlock()

	return fn
}

func (i *functionIndex) delete(ptr unsafe.Pointer) {
	i.Lock()
	delete(i.functions, ptr)
	i.Unlock()

	C.free(ptr)
}

func (i *functionIndex) destroy() {
	i.Lock()

	for ptr, _ := range i.functions {
		delete(i.functions, ptr)
		C.free(ptr)
	}
	i.Unlock()
}

type ctxIndex struct {
	sync.RWMutex
	ctxs map[unsafe.Pointer]*Context
}

func (ci *ctxIndex) add(ctx *Context) unsafe.Pointer {

	ci.RLock()
	for ptr, ctxPtr := range ci.ctxs {
		if ctxPtr == ctx {
			ci.RUnlock()
			return ptr
		}
	}
	ci.RUnlock()

	ci.Lock()
	for ptr, ctxPtr := range ci.ctxs {
		if ctxPtr == ctx {
			ci.Unlock()
			return ptr
		}
	}
	ptr := C.malloc(1)
	ci.ctxs[ptr] = ctx
	ci.Unlock()

	return ptr
}

func (ci *ctxIndex) get(ptr unsafe.Pointer) *Context {
	ci.RLock()
	ctx := ci.ctxs[ptr]
	ci.RUnlock()
	return ctx
}

func (ci *ctxIndex) delete(ctx *Context) {
	ci.Lock()
	for ptr, ctxPtr := range ci.ctxs {
		if ctxPtr == ctx {
			delete(ci.ctxs, ptr)
			C.free(ptr)
			ci.Unlock()
			return
		}
	}
	panic(fmt.Sprintf("context (%p) doesn't exist", ctx))
}

var contexts *ctxIndex

func init() {
	contexts = &ctxIndex{
		ctxs: make(map[unsafe.Pointer]*Context),
	}
}
