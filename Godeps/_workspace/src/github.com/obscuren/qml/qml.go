package qml

// #include <stdlib.h>
//
// #include "capi.h"
//
import "C"

import (
	"errors"
	"fmt"
	"gopkg.in/qml.v1/gl/glbase"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// Engine provides an environment for instantiating QML components.
type Engine struct {
	Common
	values    map[interface{}]*valueFold
	destroyed bool

	imageProviders map[string]*func(imageId string, width, height int) image.Image
}

var engines = make(map[unsafe.Pointer]*Engine)

// NewEngine returns a new QML engine.
//
// The Destory method must be called to finalize the engine and
// release any resources used.
func NewEngine() *Engine {
	engine := &Engine{values: make(map[interface{}]*valueFold)}
	RunMain(func() {
		engine.addr = C.newEngine(nil)
		engine.engine = engine
		engine.imageProviders = make(map[string]*func(imageId string, width, height int) image.Image)
		engines[engine.addr] = engine
		stats.enginesAlive(+1)
	})
	return engine
}

func (e *Engine) assertValid() {
	if e.destroyed {
		panic("engine already destroyed")
	}
}

// Destroy finalizes the engine and releases any resources used.
// The engine must not be used after calling this method.
//
// It is safe to call Destroy more than once.
func (e *Engine) Destroy() {
	if !e.destroyed {
		RunMain(func() {
			if !e.destroyed {
				e.destroyed = true
				C.delObjectLater(e.addr)
				if len(e.values) == 0 {
					delete(engines, e.addr)
				} else {
					// The engine reference keeps those values alive.
					// The last value destroyed will clear it.
				}
				stats.enginesAlive(-1)
			}
		})
	}
}

// Load loads a new component with the provided location and with the
// content read from r. The location informs the resource name for
// logged messages, and its path is used to locate any other resources
// referenced by the QML content.
//
// Once a component is loaded, component instances may be created from
// the resulting object via its Create and CreateWindow methods.
func (e *Engine) Load(location string, r io.Reader) (Object, error) {
	var cdata *C.char
	var cdatalen C.int

	qrc := strings.HasPrefix(location, "qrc:")
	if qrc {
		if r != nil {
			return nil, fmt.Errorf("cannot load qrc resource while providing data: %s", location)
		}
	} else {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if colon, slash := strings.Index(location, ":"), strings.Index(location, "/"); colon == -1 || slash <= colon {
			if filepath.IsAbs(location) {
				location = "file:///" + filepath.ToSlash(location)
			} else {
				dir, err := os.Getwd()
				if err != nil {
					return nil, fmt.Errorf("cannot obtain absolute path: %v", err)
				}
				location = "file:///" + filepath.ToSlash(filepath.Join(dir, location))
			}
		}

		// Workaround issue #84 (QTBUG-41193) by not refering to an existent file.
		if s := strings.TrimPrefix(location, "file:///"); s != location {
			if _, err := os.Stat(filepath.FromSlash(s)); err == nil {
				location = location + "."
			}
		}

		cdata, cdatalen = unsafeBytesData(data)
	}

	var err error
	cloc, cloclen := unsafeStringData(location)
	comp := &Common{engine: e}
	RunMain(func() {
		// TODO The component's parent should probably be the engine.
		comp.addr = C.newComponent(e.addr, nilPtr)
		if qrc {
			C.componentLoadURL(comp.addr, cloc, cloclen)
		} else {
			C.componentSetData(comp.addr, cdata, cdatalen, cloc, cloclen)
		}
		message := C.componentErrorString(comp.addr)
		if message != nilCharPtr {
			err = errors.New(strings.TrimRight(C.GoString(message), "\n"))
			C.free(unsafe.Pointer(message))
		}
	})
	if err != nil {
		return nil, err
	}
	return comp, nil
}

// LoadFile loads a component from the provided QML file.
// Resources referenced by the QML content will be resolved relative to its path.
//
// Once a component is loaded, component instances may be created from
// the resulting object via its Create and CreateWindow methods.
func (e *Engine) LoadFile(path string) (Object, error) {
	if strings.HasPrefix(path, "qrc:") {
		return e.Load(path, nil)
	}
	// TODO Test this.
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return e.Load(path, f)
}

// LoadString loads a component from the provided QML string.
// The location informs the resource name for logged messages, and its
// path is used to locate any other resources referenced by the QML content.
//
// Once a component is loaded, component instances may be created from
// the resulting object via its Create and CreateWindow methods.
func (e *Engine) LoadString(location, qml string) (Object, error) {
	return e.Load(location, strings.NewReader(qml))
}

// Context returns the engine's root context.
func (e *Engine) Context() *Context {
	e.assertValid()
	var ctx Context
	ctx.engine = e
	RunMain(func() {
		ctx.addr = C.engineRootContext(e.addr)
	})
	return &ctx
}

// TODO ObjectOf is probably still worth it, but turned out unnecessary
//      for GL functionality. Test it properly before introducing it.

// ObjectOf returns the QML Object representation of the provided Go value
// within the e engine.
//func (e *Engine) ObjectOf(value interface{}) Object {
//	// TODO Would be good to preserve identity on the Go side. See unpackDataValue as well.
//	return &Common{
//		engine: e,
//		addr:   wrapGoValue(e, value, cppOwner),
//	}
//}

// Painter is provided to Paint methods on Go types that have displayable content.
type Painter struct {
	engine *Engine
	obj    Object
	glctxt glbase.Context
}

// Object returns the underlying object being painted.
func (p *Painter) Object() Object {
	return p.obj
}

// GLContext returns the OpenGL context for this painter.
func (p *Painter) GLContext() *glbase.Context {
	return &p.glctxt
}

// AddImageProvider registers f to be called when an image is requested by QML code
// with the specified provider identifier. It is a runtime error to register the same
// provider identifier multiple times.
//
// The imgId provided to f is the requested image source, with the "image:" scheme
// and provider identifier removed. For example, with an image image source of
// "image://myprovider/icons/home.ext", the respective imgId would be "icons/home.ext".
//
// If either the width or the height parameters provided to f are zero, no specific
// size for the image was requested. If non-zero, the returned image should have the
// the provided size, and will be resized if the returned image has a different size.
//
// See the documentation for more details on image providers:
//
//   http://qt-project.org/doc/qt-5.0/qtquick/qquickimageprovider.html
//
func (e *Engine) AddImageProvider(prvId string, f func(imgId string, width, height int) image.Image) {
	if _, ok := e.imageProviders[prvId]; ok {
		panic(fmt.Sprintf("engine already has an image provider with id %q", prvId))
	}
	e.imageProviders[prvId] = &f
	cprvId, cprvIdLen := unsafeStringData(prvId)
	RunMain(func() {
		qprvId := C.newString(cprvId, cprvIdLen)
		defer C.delString(qprvId)
		C.engineAddImageProvider(e.addr, qprvId, unsafe.Pointer(&f))
	})
}

//export hookRequestImage
func hookRequestImage(imageFunc unsafe.Pointer, cid *C.char, cidLen, cwidth, cheight C.int) unsafe.Pointer {
	f := *(*func(imgId string, width, height int) image.Image)(imageFunc)

	id := unsafeString(cid, cidLen)
	width := int(cwidth)
	height := int(cheight)

	img := f(id, width, height)

	var cimage unsafe.Pointer

	rect := img.Bounds()
	width = rect.Max.X - rect.Min.X
	height = rect.Max.Y - rect.Min.Y
	cimage = C.newImage(C.int(width), C.int(height))

	var cbits []byte
	cbitsh := (*reflect.SliceHeader)((unsafe.Pointer)(&cbits))
	cbitsh.Data = (uintptr)((unsafe.Pointer)(C.imageBits(cimage)))
	cbitsh.Len = width * height * 4 // RGBA
	cbitsh.Cap = cbitsh.Len

	i := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			*(*uint32)(unsafe.Pointer(&cbits[i])) = (a>>8)<<24 | (r>>8)<<16 | (g>>8)<<8 | (b >> 8)
			i += 4
		}
	}
	return cimage
}

// Context represents a QML context that can hold variables visible
// to logic running within it.
type Context struct {
	Common
}

// SetVar makes the provided value available as a variable with the
// given name for QML code executed within the c context.
//
// If value is a struct, its exported fields are also made accessible to
// QML code as attributes of the named object. The attribute name in the
// object has the same name of the Go field name, except for the first
// letter which is lowercased. This is conventional and enforced by
// the QML implementation.
//
// The engine will hold a reference to the provided value, so it will
// not be garbage collected until the engine is destroyed, even if the
// value is unused or changed.
func (ctx *Context) SetVar(name string, value interface{}) {
	cname, cnamelen := unsafeStringData(name)
	RunMain(func() {
		var dvalue C.DataValue
		packDataValue(value, &dvalue, ctx.engine, cppOwner)

		qname := C.newString(cname, cnamelen)
		defer C.delString(qname)

		C.contextSetProperty(ctx.addr, qname, &dvalue)
	})
}

// SetVars makes the exported fields of the provided value available as
// variables for QML code executed within the c context. The variable names
// will have the same name of the Go field names, except for the first
// letter which is lowercased. This is conventional and enforced by
// the QML implementation.
//
// The engine will hold a reference to the provided value, so it will
// not be garbage collected until the engine is destroyed, even if the
// value is unused or changed.
func (ctx *Context) SetVars(value interface{}) {
	RunMain(func() {
		C.contextSetObject(ctx.addr, wrapGoValue(ctx.engine, value, cppOwner))
	})
}

// Var returns the context variable with the given name.
func (ctx *Context) Var(name string) interface{} {
	cname, cnamelen := unsafeStringData(name)

	var dvalue C.DataValue
	RunMain(func() {
		qname := C.newString(cname, cnamelen)
		defer C.delString(qname)

		C.contextGetProperty(ctx.addr, qname, &dvalue)
	})
	return unpackDataValue(&dvalue, ctx.engine)
}

// Spawn creates a new context that has ctx as a parent.
func (ctx *Context) Spawn() *Context {
	var result Context
	result.engine = ctx.engine
	RunMain(func() {
		result.addr = C.contextSpawn(ctx.addr)
	})
	return &result
}

// Object is the common interface implemented by all QML types.
//
// See the documentation of Common for details about this interface.
type Object interface {
	Common() *Common
	Addr() uintptr
	TypeName() string
	Interface() interface{}
	Set(property string, value interface{})
	Property(name string) interface{}
	Int(property string) int
	Int64(property string) int64
	Float64(property string) float64
	Bool(property string) bool
	String(property string) string
	Color(property string) color.RGBA
	Object(property string) Object
	Map(property string) *Map
	List(property string) *List
	ObjectByName(objectName string) Object
	Call(method string, params ...interface{}) interface{}
	Create(ctx *Context) Object
	CreateWindow(ctx *Context) *Window
	Destroy()
	On(signal string, function interface{})
}

// List holds a QML list which may be converted to a Go slice of an
// appropriate type via Convert.
//
// In the future this will also be able to hold a reference
// to QML-owned maps, so they can be mutated in place.
type List struct {
	// In the future this will be able to hold a reference to QML-owned
	// lists, so they can be mutated.
	data []interface{}
}

// Len returns the number of elements in the list.
func (l *List) Len() int {
	return len(l.data)
}

// Convert allocates a new slice and copies the list content into it,
// performing type conversions as possible, and then assigns the result
// to the slice pointed to by sliceAddr.
// Convert panics if the list values are not compatible with the
// provided slice.
func (l *List) Convert(sliceAddr interface{}) {
	toPtr := reflect.ValueOf(sliceAddr)
	if toPtr.Kind() != reflect.Ptr || toPtr.Type().Elem().Kind() != reflect.Slice {
		panic(fmt.Sprintf("List.Convert got a sliceAddr parameter that is not a slice address: %#v", sliceAddr))
	}
	err := convertAndSet(toPtr.Elem(), reflect.ValueOf(l), reflect.Value{})
	if err != nil {
		panic(err.Error())
	}
}

// Map holds a QML map which may be converted to a Go map of an
// appropriate type via Convert.
//
// In the future this will also be able to hold a reference
// to QML-owned maps, so they can be mutated in place.
type Map struct {
	data []interface{}
}

// Len returns the number of pairs in the map.
func (m *Map) Len() int {
	return len(m.data) / 2
}

// Convert allocates a new map and copies the content of m property to it,
// performing type conversions as possible, and then assigns the result to
// the map pointed to by mapAddr. Map panics if m contains values that
// cannot be converted to the type of the map at mapAddr.
func (m *Map) Convert(mapAddr interface{}) {
	toPtr := reflect.ValueOf(mapAddr)
	if toPtr.Kind() != reflect.Ptr || toPtr.Type().Elem().Kind() != reflect.Map {
		panic(fmt.Sprintf("Map.Convert got a mapAddr parameter that is not a map address: %#v", mapAddr))
	}
	err := convertAndSet(toPtr.Elem(), reflect.ValueOf(m), reflect.Value{})
	if err != nil {
		panic(err.Error())
	}
}

// Common implements the common behavior of all QML objects.
// It implements the Object interface.
type Common struct {
	addr   unsafe.Pointer
	engine *Engine
}

var _ Object = (*Common)(nil)

// CommonOf returns the Common QML value for the QObject at addr.
//
// This is meant for extensions that integrate directly with the
// underlying QML logic.
func CommonOf(addr unsafe.Pointer, engine *Engine) *Common {
	return &Common{addr, engine}
}

// Common returns obj itself.
//
// This provides access to the underlying *Common for types that
// embed it, when these are used via the Object interface.
func (obj *Common) Common() *Common {
	return obj
}

// TypeName returns the underlying type name for the held value.
func (obj *Common) TypeName() string {
	var name string
	RunMain(func() {
		name = C.GoString(C.objectTypeName(obj.addr))
	})
	return name
}

// Addr returns the QML object address.
//
// This is meant for extensions that integrate directly with the
// underlying QML logic.
func (obj *Common) Addr() uintptr {
	return uintptr(obj.addr)
}

// Interface returns the underlying Go value that is being held by
// the object wrapper.
//
// It is a runtime error to call Interface on values that are not
// backed by a Go value.
func (obj *Common) Interface() interface{} {
	var result interface{}
	var cerr *C.error
	RunMain(func() {
		var fold *valueFold
		if cerr = C.objectGoAddr(obj.addr, (*unsafe.Pointer)(unsafe.Pointer(&fold))); cerr == nil {
			result = fold.gvalue
		}
	})
	cmust(cerr)
	return result
}

// Set changes the named object property to the given value.
func (obj *Common) Set(property string, value interface{}) {
	cproperty := C.CString(property)
	defer C.free(unsafe.Pointer(cproperty))
	var cerr *C.error
	RunMain(func() {
		var dvalue C.DataValue
		packDataValue(value, &dvalue, obj.engine, cppOwner)
		cerr = C.objectSetProperty(obj.addr, cproperty, &dvalue)
	})
	cmust(cerr)
}

// Property returns the current value for a property of the object.
// If the property type is known, type-specific methods such as Int
// and String are more convenient to use.
// Property panics if the property does not exist.
func (obj *Common) Property(name string) interface{} {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	var dvalue C.DataValue
	var found C.int
	RunMain(func() {
		found = C.objectGetProperty(obj.addr, cname, &dvalue)
	})
	if found == 0 {
		panic(fmt.Sprintf("object does not have a %q property", name))
	}
	return unpackDataValue(&dvalue, obj.engine)
}

// Int returns the int value of the named property.
// Int panics if the property cannot be represented as an int.
func (obj *Common) Int(property string) int {
	switch value := obj.Property(property).(type) {
	case int64:
		return int(value)
	case int:
		return value
	case uint64:
		return int(value)
	case uint32:
		return int(value)
	case uintptr:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	default:
		panic(fmt.Sprintf("value of property %q cannot be represented as an int: %#v", property, value))
	}
}

// Int64 returns the int64 value of the named property.
// Int64 panics if the property cannot be represented as an int64.
func (obj *Common) Int64(property string) int64 {
	switch value := obj.Property(property).(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case uint64:
		return int64(value)
	case uint32:
		return int64(value)
	case uintptr:
		return int64(value)
	case float32:
		return int64(value)
	case float64:
		return int64(value)
	default:
		panic(fmt.Sprintf("value of property %q cannot be represented as an int64: %#v", property, value))
	}
}

// Float64 returns the float64 value of the named property.
// Float64 panics if the property cannot be represented as float64.
func (obj *Common) Float64(property string) float64 {
	switch value := obj.Property(property).(type) {
	case int64:
		return float64(value)
	case int:
		return float64(value)
	case uint64:
		return float64(value)
	case uint32:
		return float64(value)
	case uintptr:
		return float64(value)
	case float32:
		return float64(value)
	case float64:
		return value
	default:
		panic(fmt.Sprintf("value of property %q cannot be represented as a float64: %#v", property, value))
	}
}

// Bool returns the bool value of the named property.
// Bool panics if the property is not a bool.
func (obj *Common) Bool(property string) bool {
	value := obj.Property(property)
	if b, ok := value.(bool); ok {
		return b
	}
	panic(fmt.Sprintf("value of property %q is not a bool: %#v", property, value))
}

// String returns the string value of the named property.
// String panics if the property is not a string.
func (obj *Common) String(property string) string {
	value := obj.Property(property)
	if s, ok := value.(string); ok {
		return s
	}
	panic(fmt.Sprintf("value of property %q is not a string: %#v", property, value))
}

// Color returns the RGBA value of the named property.
// Color panics if the property is not a color.
func (obj *Common) Color(property string) color.RGBA {
	value := obj.Property(property)
	c, ok := value.(color.RGBA)
	if !ok {
		panic(fmt.Sprintf("value of property %q is not a color: %#v", property, value))
	}
	return c
}

// Object returns the object value of the named property.
// Object panics if the property is not a QML object.
func (obj *Common) Object(property string) Object {
	value := obj.Property(property)
	object, ok := value.(Object)
	if !ok {
		panic(fmt.Sprintf("value of property %q is not a QML object: %#v", property, value))
	}
	return object
}

// List returns the list value of the named property.
// List panics if the property is not a list.
func (obj *Common) List(property string) *List {
	value := obj.Property(property)
	m, ok := value.(*List)
	if !ok {
		panic(fmt.Sprintf("value of property %q is not a QML list: %#v", property, value))
	}
	return m
}

// Map returns the map value of the named property.
// Map panics if the property is not a map.
func (obj *Common) Map(property string) *Map {
	value := obj.Property(property)
	m, ok := value.(*Map)
	if !ok {
		panic(fmt.Sprintf("value of property %q is not a QML map: %#v", property, value))
	}
	return m
}

// ObjectByName returns the Object value of the descendant object that
// was defined with the objectName property set to the provided value.
// ObjectByName panics if the object is not found.
func (obj *Common) ObjectByName(objectName string) Object {
	cname, cnamelen := unsafeStringData(objectName)
	var dvalue C.DataValue
	var object Object
	RunMain(func() {
		qname := C.newString(cname, cnamelen)
		defer C.delString(qname)
		C.objectFindChild(obj.addr, qname, &dvalue)
		// unpackDataValue will also initialize the Go type, if necessary.
		value := unpackDataValue(&dvalue, obj.engine)
		if dvalue.dataType == C.DTGoAddr {
			datap := unsafe.Pointer(&dvalue.data)
			fold := (*(**valueFold)(datap))
			if fold.init.IsValid() {
				panic("internal error: custom Go type not initialized")
			}
			object = &Common{fold.cvalue, fold.engine}
		} else {
			object, _ = value.(Object)
		}
	})
	if object == nil {
		panic(fmt.Sprintf("cannot find descendant with objectName == %q", objectName))
	}
	return object
}

// Call calls the given object method with the provided parameters.
// Call panics if the method does not exist.
func (obj *Common) Call(method string, params ...interface{}) interface{} {
	if len(params) > len(dataValueArray) {
		panic("too many parameters")
	}
	cmethod, cmethodLen := unsafeStringData(method)
	var result C.DataValue
	var cerr *C.error
	RunMain(func() {
		for i, param := range params {
			packDataValue(param, &dataValueArray[i], obj.engine, jsOwner)
		}
		cerr = C.objectInvoke(obj.addr, cmethod, cmethodLen, &result, &dataValueArray[0], C.int(len(params)))
	})
	cmust(cerr)
	return unpackDataValue(&result, obj.engine)
}

// Create creates a new instance of the component held by obj.
// The component instance runs under the ctx context. If ctx is nil,
// it runs under the same context as obj.
//
// The Create method panics if called on an object that does not
// represent a QML component.
func (obj *Common) Create(ctx *Context) Object {
	if C.objectIsComponent(obj.addr) == 0 {
		panic("object is not a component")
	}
	var root Common
	root.engine = obj.engine
	RunMain(func() {
		ctxaddr := nilPtr
		if ctx != nil {
			ctxaddr = ctx.addr
		}
		root.addr = C.componentCreate(obj.addr, ctxaddr)
	})
	return &root
}

// CreateWindow creates a new instance of the component held by obj,
// and creates a new window holding the instance as its root object.
// The component instance runs under the ctx context. If ctx is nil,
// it runs under the same context as obj.
//
// The CreateWindow method panics if called on an object that
// does not represent a QML component.
func (obj *Common) CreateWindow(ctx *Context) *Window {
	if C.objectIsComponent(obj.addr) == 0 {
		panic("object is not a component")
	}
	var win Window
	win.engine = obj.engine
	RunMain(func() {
		ctxaddr := nilPtr
		if ctx != nil {
			ctxaddr = ctx.addr
		}
		win.addr = C.componentCreateWindow(obj.addr, ctxaddr)
	})
	return &win
}

// Destroy finalizes the value and releases any resources used.
// The value must not be used after calling this method.
func (obj *Common) Destroy() {
	// TODO We might hook into the destroyed signal, and prevent this object
	//      from being used in post-destruction crash-prone ways.
	RunMain(func() {
		if obj.addr != nilPtr {
			C.delObjectLater(obj.addr)
			obj.addr = nilPtr
		}
	})
}

var connectedFunction = make(map[*interface{}]bool)

// On connects the named signal from obj with the provided function, so that
// when obj next emits that signal, the function is called with the parameters
// the signal carries.
//
// The provided function must accept a number of parameters that is equal to
// or less than the number of parameters provided by the signal, and the
// resepctive parameter types must match exactly or be conversible according
// to normal Go rules.
//
// For example:
//
//     obj.On("clicked", func() { fmt.Println("obj got a click") })
//
// Note that Go uses the real signal name, rather than the one used when
// defining QML signal handlers ("clicked" rather than "onClicked").
//
// For more details regarding signals and QML see:
//
//     http://qt-project.org/doc/qt-5.0/qtqml/qml-qtquick2-connections.html
//
func (obj *Common) On(signal string, function interface{}) {
	funcv := reflect.ValueOf(function)
	funct := funcv.Type()
	if funcv.Kind() != reflect.Func {
		panic("function provided to On is not a function or method")
	}
	if funct.NumIn() > C.MaxParams {
		panic("function takes too many arguments")
	}
	csignal, csignallen := unsafeStringData(signal)
	var cerr *C.error
	RunMain(func() {
		cerr = C.objectConnect(obj.addr, csignal, csignallen, obj.engine.addr, unsafe.Pointer(&function), C.int(funcv.Type().NumIn()))
		if cerr == nil {
			connectedFunction[&function] = true
			stats.connectionsAlive(+1)
		}
	})
	cmust(cerr)
}

//export hookSignalDisconnect
func hookSignalDisconnect(funcp unsafe.Pointer) {
	before := len(connectedFunction)
	delete(connectedFunction, (*interface{})(funcp))
	if before == len(connectedFunction) {
		panic("disconnecting unknown signal function")
	}
	stats.connectionsAlive(-1)
}

//export hookSignalCall
func hookSignalCall(enginep unsafe.Pointer, funcp unsafe.Pointer, args *C.DataValue) {
	engine := engines[enginep]
	if engine == nil {
		panic("signal called after engine was destroyed")
	}
	funcv := reflect.ValueOf(*(*interface{})(funcp))
	funct := funcv.Type()
	numIn := funct.NumIn()
	var params [C.MaxParams]reflect.Value
	for i := 0; i < numIn; i++ {
		arg := (*C.DataValue)(unsafe.Pointer(uintptr(unsafe.Pointer(args)) + uintptr(i)*dataValueSize))
		param := reflect.ValueOf(unpackDataValue(arg, engine))
		if paramt := funct.In(i); param.Type() != paramt {
			// TODO Provide a better error message when this fails.
			param = param.Convert(paramt)
		}
		params[i] = param
	}
	funcv.Call(params[:numIn])
}

func cerror(cerr *C.error) error {
	err := errors.New(C.GoString((*C.char)(unsafe.Pointer(cerr))))
	C.free(unsafe.Pointer(cerr))
	return err
}

func cmust(cerr *C.error) {
	if cerr != nil {
		panic(cerror(cerr).Error())
	}
}

// TODO Signal emitting support for go values.

// Window represents a QML window where components are rendered.
type Window struct {
	Common
}

// Show exposes the window.
func (win *Window) Show() {
	RunMain(func() {
		C.windowShow(win.addr)
	})
}

// Hide hides the window.
func (win *Window) Hide() {
	RunMain(func() {
		C.windowHide(win.addr)
	})
}

// PlatformId returns the window's platform id.
//
// For platforms where this id might be useful, the value returned will
// uniquely represent the window inside the corresponding screen.
func (win *Window) PlatformId() uintptr {
	var id uintptr
	RunMain(func() {
		id = uintptr(C.windowPlatformId(win.addr))
	})
	return id
}

// Root returns the root object being rendered.
//
// If the window was defined in QML code, the root object is the window itself.
func (win *Window) Root() Object {
	var obj Common
	obj.engine = win.engine
	RunMain(func() {
		obj.addr = C.windowRootObject(win.addr)
	})
	return &obj
}

// Wait blocks the current goroutine until the window is closed.
func (win *Window) Wait() {
	// XXX Test this.
	var m sync.Mutex
	m.Lock()
	RunMain(func() {
		// TODO Must be able to wait for the same Window from multiple goroutines.
		// TODO If the window is not visible, must return immediately.
		waitingWindows[win.addr] = &m
		C.windowConnectHidden(win.addr)
	})
	m.Lock()
}

var waitingWindows = make(map[unsafe.Pointer]*sync.Mutex)

//export hookWindowHidden
func hookWindowHidden(addr unsafe.Pointer) {
	m, ok := waitingWindows[addr]
	if !ok {
		panic("window is not waiting")
	}
	delete(waitingWindows, addr)
	m.Unlock()
}

// Snapshot returns an image with the visible contents of the window.
// The main GUI thread is paused while the data is being acquired.
func (win *Window) Snapshot() image.Image {
	// TODO Test this.
	var cimage unsafe.Pointer
	RunMain(func() {
		cimage = C.windowGrabWindow(win.addr)
	})
	defer C.delImage(cimage)

	// This should be safe to be done out of the main GUI thread.
	var cwidth, cheight C.int
	C.imageSize(cimage, &cwidth, &cheight)

	var cbits []byte
	cbitsh := (*reflect.SliceHeader)((unsafe.Pointer)(&cbits))
	cbitsh.Data = (uintptr)((unsafe.Pointer)(C.imageConstBits(cimage)))
	cbitsh.Len = int(cwidth * cheight * 8) // ARGB
	cbitsh.Cap = cbitsh.Len

	image := image.NewRGBA(image.Rect(0, 0, int(cwidth), int(cheight)))
	l := int(cwidth * cheight * 4)
	for i := 0; i < l; i += 4 {
		var c uint32 = *(*uint32)(unsafe.Pointer(&cbits[i]))
		image.Pix[i+0] = byte(c >> 16)
		image.Pix[i+1] = byte(c >> 8)
		image.Pix[i+2] = byte(c)
		image.Pix[i+3] = byte(c >> 24)
	}
	return image
}

// TypeSpec holds the specification of a QML type that is backed by Go logic.
//
// The type specification must be registered with the RegisterTypes function
// before it will be visible to QML code, as in:
//
//     qml.RegisterTypes("GoExtensions", 1, 0, []qml.TypeSpec{{
//		Init: func(p *Person, obj qml.Object) {},
//     }})
//
// See the package documentation for more details.
//
type TypeSpec struct {
	// Init must be set to a function that is called when QML code requests
	// the creation of a new value of this type. The provided function must
	// have the following type:
	//
	//     func(value *CustomType, object qml.Object)
	//
	// Where CustomType is the custom type being registered. The function will
	// be called with a newly created *CustomType and its respective qml.Object.
	Init interface{}

	// Name optionally holds the identifier the type is known as within QML code,
	// when the registered extension module is imported. If not specified, the
	// name of the Go type provided as the first argument of Init is used instead.
	Name string

	// Singleton defines whether a single instance of the type should be used
	// for all accesses, as a singleton value. If true, all properties of the
	// singleton value are directly accessible under the type name.
	Singleton bool

	private struct{} // Force use of fields by name.
}

var types []*TypeSpec

// RegisterTypes registers the provided list of type specifications for use
// by QML code. To access the registered types, they must be imported from the
// provided location and major.minor version numbers.
//
// For example, with a location "GoExtensions", major 4, and minor 2, this statement
// imports all the registered types in the module's namespace:
//
//     import GoExtensions 4.2
//
// See the documentation on QML import statements for details on these:
//
//     http://qt-project.org/doc/qt-5.0/qtqml/qtqml-syntax-imports.html
//
func RegisterTypes(location string, major, minor int, types []TypeSpec) {
	for i := range types {
		err := registerType(location, major, minor, &types[i])
		if err != nil {
			panic(err)
		}
	}
}

func registerType(location string, major, minor int, spec *TypeSpec) error {
	// Copy and hold a reference to the spec data.
	localSpec := *spec

	f := reflect.ValueOf(localSpec.Init)
	ft := f.Type()
	if ft.Kind() != reflect.Func {
		return fmt.Errorf("TypeSpec.Init must be a function, got %#v", localSpec.Init)
	}
	if ft.NumIn() != 2 {
		return fmt.Errorf("TypeSpec.Init's function must accept two arguments: %s", ft)
	}
	firstArg := ft.In(0)
	if firstArg.Kind() != reflect.Ptr || firstArg.Elem().Kind() == reflect.Ptr {
		return fmt.Errorf("TypeSpec.Init's function must take a pointer type as the second argument: %s", ft)
	}
	if ft.In(1) != typeObject {
		return fmt.Errorf("TypeSpec.Init's function must take qml.Object as the second argument: %s", ft)
	}
	customType := typeInfo(reflect.New(firstArg.Elem()).Interface())
	if localSpec.Name == "" {
		localSpec.Name = firstArg.Elem().Name()
		if localSpec.Name == "" {
			panic("cannot determine registered type name; please provide one explicitly")
		}
	}

	var err error
	RunMain(func() {
		cloc := C.CString(location)
		cname := C.CString(localSpec.Name)
		cres := C.int(0)
		if localSpec.Singleton {
			cres = C.registerSingleton(cloc, C.int(major), C.int(minor), cname, customType, unsafe.Pointer(&localSpec))
		} else {
			cres = C.registerType(cloc, C.int(major), C.int(minor), cname, customType, unsafe.Pointer(&localSpec))
		}
		// It doesn't look like it keeps references to these, but it's undocumented and unclear.
		C.free(unsafe.Pointer(cloc))
		C.free(unsafe.Pointer(cname))
		if cres == -1 {
			err = fmt.Errorf("QML engine failed to register type; invalid type location or name?")
		} else {
			types = append(types, &localSpec)
		}
	})

	return err
}

// RegisterConverter registers the convereter function to be called when a
// value with the provided type name is obtained from QML logic. The function
// must return the new value to be used in place of the original value.
func RegisterConverter(typeName string, converter func(engine *Engine, obj Object) interface{}) {
	if converter == nil {
		delete(converters, typeName)
	} else {
		converters[typeName] = converter
	}
}

var converters = make(map[string]func(engine *Engine, obj Object) interface{})

// LoadResources registers all resources in the provided resources collection,
// making them available to be loaded by any Engine and QML file.
// Registered resources are made available under "qrc:///some/path", where
// "some/path" is the path the resource was added with.
func LoadResources(r *Resources) {
	var base unsafe.Pointer
	if len(r.sdata) > 0 {
		base = *(*unsafe.Pointer)(unsafe.Pointer(&r.sdata))
	} else if len(r.bdata) > 0 {
		base = *(*unsafe.Pointer)(unsafe.Pointer(&r.bdata))
	}
	tree := (*C.char)(unsafe.Pointer(uintptr(base)+uintptr(r.treeOffset)))
	name := (*C.char)(unsafe.Pointer(uintptr(base)+uintptr(r.nameOffset)))
	data := (*C.char)(unsafe.Pointer(uintptr(base)+uintptr(r.dataOffset)))
	C.registerResourceData(C.int(r.version), tree, name, data)
}

// UnloadResources unregisters all previously registered resources from r.
func UnloadResources(r *Resources) {
	var base unsafe.Pointer
	if len(r.sdata) > 0 {
		base = *(*unsafe.Pointer)(unsafe.Pointer(&r.sdata))
	} else if len(r.bdata) > 0 {
		base = *(*unsafe.Pointer)(unsafe.Pointer(&r.bdata))
	}
	tree := (*C.char)(unsafe.Pointer(uintptr(base)+uintptr(r.treeOffset)))
	name := (*C.char)(unsafe.Pointer(uintptr(base)+uintptr(r.nameOffset)))
	data := (*C.char)(unsafe.Pointer(uintptr(base)+uintptr(r.dataOffset)))
	C.unregisterResourceData(C.int(r.version), tree, name, data)
}
