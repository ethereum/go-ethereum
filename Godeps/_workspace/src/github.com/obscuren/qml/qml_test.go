package qml_test

import (
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"gopkg.in/qml.v1"
	"gopkg.in/qml.v1/cpptest"
	"gopkg.in/qml.v1/gl/2.0"
	"path/filepath"
)

func init() { qml.SetupTesting() }

func Test(t *testing.T) { TestingT(t) }

type S struct {
	engine  *qml.Engine
	context *qml.Context
}

var _ = Suite(&S{})

func (s *S) SetUpTest(c *C) {
	qml.SetLogger(c)
	qml.CollectStats(true)
	qml.ResetStats()

	stats := qml.Stats()
	if stats.EnginesAlive > 0 || stats.ValuesAlive > 0 || stats.ConnectionsAlive > 0 {
		panic(fmt.Sprintf("Test started with values alive: %#v\n", stats))
	}

	s.engine = qml.NewEngine()
	s.context = s.engine.Context()
}

func (s *S) TearDownTest(c *C) {
	s.engine.Destroy()

	retries := 30 // Three seconds top.
	for {
		// Do not call qml.Flush here. It creates a nested event loop
		// that attempts to process the deferred object deletes and cannot,
		// because deferred deletes are only processed at the same loop level.
		// So it *reposts* the deferred deletion event, in practice *preventing*
		// these objects from being deleted.
		runtime.GC()
		stats := qml.Stats()
		if stats.EnginesAlive == 0 && stats.ValuesAlive == 0 && stats.ConnectionsAlive == 0 {
			break
		}
		if retries == 0 {
			panic(fmt.Sprintf("there are values alive:\n%#v\n", stats))
		}
		retries--
		time.Sleep(100 * time.Millisecond)
		if retries%10 == 0 {
			c.Logf("There are still objects alive; waiting for them to die: %#v\n", stats)
		}
	}

	qml.SetLogger(nil)
}

type GoRect struct {
	PaintCount int
}

func (r *GoRect) Paint(p *qml.Painter) {
	r.PaintCount++

	obj := p.Object()

	gl := GL.API(p)

	width := float32(obj.Int("width"))
	height := float32(obj.Int("height"))

	gl.Color3f(1.0, 0.0, 0.0)
	gl.Begin(GL.QUADS)
	gl.Vertex2f(0, 0)
	gl.Vertex2f(width, 0)
	gl.Vertex2f(width, height)
	gl.Vertex2f(0, height)
	gl.End()
}

type GoType struct {
	private bool // Besides being private, also adds a gap in the reflect field index.

	StringValue     string
	StringAddrValue *string
	BoolValue       bool
	IntValue        int
	Int64Value      int64
	Int32Value      int32
	Uint32Value     uint32
	Float64Value    float64
	Float32Value    float32
	AnyValue        interface{}
	ObjectValue     qml.Object
	ColorValue      color.RGBA
	IntsValue       []int
	ObjectsValue    []qml.Object
	MapValue        map[string]interface{}

	SetterStringValue  string
	SetterObjectsValue []qml.Object

	setterStringValueChanged  int
	setterStringValueSet      string
	setterObjectsValueChanged int
	setterObjectsValueSet     []qml.Object

	getterStringValue        string
	getterStringValueChanged int

	// The object representing this value, on custom type tests.
	object qml.Object
}

// Force a gap in the reflect method index and ensure the handling
// of private methods is being done properly.
func (ts *GoType) privateMethod() {}

func (ts *GoType) StringMethod() string {
	return ts.StringValue
}

func (ts *GoType) SetSetterStringValue(s string) {
	ts.setterStringValueChanged++
	ts.setterStringValueSet = s
}

func (ts *GoType) SetSetterObjectsValue(v []qml.Object) {
	ts.setterObjectsValueChanged++
	ts.setterObjectsValueSet = v
}

func (ts *GoType) GetterStringValue() string {
	return ts.getterStringValue
}

func (ts *GoType) SetGetterStringValue(s string) {
	ts.getterStringValueChanged++
	ts.getterStringValue = s
}

func (ts *GoType) SetMapValue(m map[string]interface{}) {
	ts.MapValue = m
}

func (ts *GoType) Mod(dividend, divisor int) (int, error) {
	if divisor == 0 {
		return 0, fmt.Errorf("<division by zero>")
	}
	return dividend % divisor, nil
}

func (ts *GoType) ChangeString(new string) (old string) {
	old = ts.StringValue
	ts.StringValue = new
	return
}

func (ts *GoType) NotifyStringChanged() {
	qml.Changed(ts, &ts.StringValue)
}

func (ts *GoType) IncrementInt() {
	ts.IntValue++
}

func (s *S) TestEngineDestroyedUse(c *C) {
	s.engine.Destroy()
	s.engine.Destroy()
	c.Assert(s.engine.Context, PanicMatches, "engine already destroyed")
}

var same = "<same>"

var getSetTests = []struct{ set, get interface{} }{
	{"value", same},
	{true, same},
	{false, same},
	{int(42), same},
	{int32(42), int(42)},
	{int64(42), same},
	{uint32(42), same},
	{uint64(42), same},
	{float64(42), same},
	{float32(42), same},
	{new(GoType), same},
	{nil, same},
	{42, same},
}

func (s *S) TestContextGetSet(c *C) {
	for i, t := range getSetTests {
		want := t.get
		if t.get == same {
			want = t.set
		}
		s.context.SetVar("key", t.set)
		c.Assert(s.context.Var("key"), Equals, want,
			Commentf("entry %d is {%v (%T), %v (%T)}", i, t.set, t.set, t.get, t.get))
	}
}

func (s *S) TestContextGetMissing(c *C) {
	c.Assert(s.context.Var("missing"), Equals, nil)
}

func (s *S) TestContextSetVars(c *C) {
	component, err := s.engine.LoadString("file.qml", "import QtQuick 2.0\nItem { width: 42 }")
	c.Assert(err, IsNil)
	root := component.Create(nil)

	vars := GoType{
		StringValue:  "<content>",
		BoolValue:    true,
		IntValue:     42,
		Int64Value:   42,
		Int32Value:   42,
		Float64Value: 4.2,
		Float32Value: 4.2,
		AnyValue:     nil,
		ObjectValue:  root,
	}
	s.context.SetVars(&vars)

	c.Assert(s.context.Var("stringValue"), Equals, "<content>")
	c.Assert(s.context.Var("boolValue"), Equals, true)
	c.Assert(s.context.Var("intValue"), Equals, 42)
	c.Assert(s.context.Var("int64Value"), Equals, int64(42))
	c.Assert(s.context.Var("int32Value"), Equals, 42)
	c.Assert(s.context.Var("float64Value"), Equals, float64(4.2))
	c.Assert(s.context.Var("float32Value"), Equals, float32(4.2))
	c.Assert(s.context.Var("anyValue"), Equals, nil)

	vars.AnyValue = 42
	c.Assert(s.context.Var("anyValue"), Equals, 42)

	c.Assert(s.context.Var("objectValue").(qml.Object).Int("width"), Equals, 42)
}

func (s *S) TestComponentSetDataError(c *C) {
	_, err := s.engine.LoadString("file.qml", "Item{}")
	c.Assert(err, ErrorMatches, "file:.*/file.qml:1 Item is not a type")
}

func (s *S) TestComponentCreateWindow(c *C) {
	data := `
		import QtQuick 2.0
		Item { width: 300; height: 200; }
	`
	component, err := s.engine.LoadString("file.qml", data)
	c.Assert(err, IsNil)

	// TODO How to test this more effectively?
	window := component.CreateWindow(nil)
	window.Show()

	// Just a smoke test, as there isn't much to assert.
	c.Assert(window.PlatformId(), Not(Equals), uintptr(0))

	// Qt doesn't hide the Window if we call it too quickly. :-(
	time.Sleep(100 * time.Millisecond)
	window.Hide()
}

func (s *S) TestContextSpawn(c *C) {
	context1 := s.engine.Context()
	context2 := context1.Spawn()

	context1.SetVar("mystr", "context1")
	context2.SetVar("mystr", "context2")

	data := `
		import QtQuick 2.0
		Item { property var s: mystr }
	`
	component, err := s.engine.LoadString("file.qml", data)
	c.Assert(err, IsNil)

	obj1 := component.Create(context1)
	obj2 := component.Create(context2)

	c.Assert(obj1.String("s"), Equals, "context1")
	c.Assert(obj2.String("s"), Equals, "context2")
}

func (s *S) TestReadVoidAddrProperty(c *C) {
	obj := cpptest.NewTestType(s.engine)
	addr := obj.Property("voidAddr").(uintptr)
	c.Assert(addr, Equals, uintptr(42))
}

func (s *S) TestRegisterConverterPlainObject(c *C) {
	qml.RegisterConverter("PlainTestType", func(engine *qml.Engine, obj qml.Object) interface{} {
		c.Check(engine, Equals, s.engine)
		c.Check(obj.String("plainType"), Matches, "(const )?PlainTestType[&*]?")
		c.Check(obj.Property("plainAddr"), FitsTypeOf, uintptr(0))
		c.Check(cpptest.PlainTestTypeN(obj), Equals, 42)
		return "<converted>"
	})
	obj := cpptest.NewTestType(s.engine)
	defer obj.Destroy()

	var calls int
	obj.On("plainEmittedCpy", func(s string) {
		c.Check(s, Equals, "<converted>")
		calls++
	})
	obj.On("plainEmittedRef", func(s string) {
		c.Check(s, Equals, "<converted>")
		calls++
	})
	obj.On("plainEmittedPtr", func(s string) {
		c.Check(s, Equals, "<converted>")
		calls++
	})
	obj.Call("emitPlain")
	c.Assert(calls, Equals, 3)
}

func (s *S) TestIssue84(c *C) {
	// Regression test for issue #84 (QTBUG-41193).
	data := `
		import QtQuick 2.0
		Item {
			id: item
			property string s1: "<before>"
			property string s2: "<after>"
			states: State {
				name: "after";
				PropertyChanges { target: item; s1: s2 }
			}
			Component.onCompleted: state = "after"
		}
	`
	filename := filepath.Join(c.MkDir(), "file.qml")
	err := ioutil.WriteFile(filename, []byte(data), 0644)
	c.Assert(err, IsNil)

	component, err := s.engine.LoadString(filename, data)
	c.Assert(err, IsNil)

	root := component.Create(nil)
	defer root.Destroy()

	c.Assert(root.String("s1"), Equals, "<after>")
}

func (s *S) TestResources(c *C) {
	var rp qml.ResourcesPacker
	rp.Add("sub/path/Foo.qml", []byte("import QtQuick 2.0\nItem { Component.onCompleted: console.log('<Foo>') }"))
	rp.AddString("sub/path/Bar.qml", "import QtQuick 2.0\nItem { Component.onCompleted: console.log('<Bar>') }")
	rp.AddString("/sub/Main.qml", "import QtQuick 2.0\nimport \"./path\"\nItem {\nFoo{}\nBar{}\n}")

	r := rp.Pack()
	qml.LoadResources(r)
	testResourcesLoaded(c, true)
	qml.UnloadResources(r)
	testResourcesLoaded(c, false)

	data := r.Bytes()

	rb, err := qml.ParseResources(data)
	c.Assert(err, IsNil)
	qml.LoadResources(rb)
	testResourcesLoaded(c, true)
	qml.UnloadResources(rb)
	testResourcesLoaded(c, false)

	rs, err := qml.ParseResourcesString(string(data))
	c.Assert(err, IsNil)
	qml.LoadResources(rs)
	testResourcesLoaded(c, true)
	qml.UnloadResources(rs)
	testResourcesLoaded(c, false)
}

func testResourcesLoaded(c *C, loaded bool) {
	engine := qml.NewEngine()
	defer engine.Destroy()
	component, err := engine.LoadFile("qrc:///sub/Main.qml")
	if loaded {
		c.Assert(err, IsNil)
	} else {
		c.Assert(err, ErrorMatches, "qrc:///sub/Main.qml:-1 File not found")
		return
	}
	root := component.Create(nil)
	defer root.Destroy()
	c.Assert(c.GetTestLog(), Matches, "(?s).*(<Foo>.*<Bar>|<Bar>.*<Foo>).*")
}

func (s *S) TestResourcesIssue107(c *C) {
	var rp qml.ResourcesPacker

	rp.Add("a/Foo.qml", []byte("import QtQuick 2.0\nItem { Component.onCompleted: console.log('<Foo>') }"))
	rp.Add("b/Bar.qml", []byte("import QtQuick 2.0\nItem { Component.onCompleted: console.log('<Bar>') }"))
	rp.Add("c/Baz.qml", []byte("import QtQuick 2.0\nItem { Component.onCompleted: console.log('<Baz>') }"))
	rp.Add("d/Buz.qml", []byte("import QtQuick 2.0\nItem { Component.onCompleted: console.log('<Buz>') }"))

	r := rp.Pack()
	qml.LoadResources(r)

	for _, name := range []string{"a/Foo", "b/Bar", "c/Baz", "d/Buz"} {
		component, err := s.engine.LoadFile("qrc:///" + name + ".qml")
		c.Assert(err, IsNil)
		root := component.Create(nil)
		defer root.Destroy()
	}
	c.Assert(c.GetTestLog(), Matches, "(?s).*<Foo>.*<Bar>.*<Baz>.*<Buz>.*")
}

type TestData struct {
	*C
	engine           *qml.Engine
	context          *qml.Context
	component        qml.Object
	root             qml.Object
	value            *GoType
	createdValue     []*GoType
	createdRect      []*GoRect
	createdSingleton []*GoType
}

var tests = []struct {
	Summary string
	Value   GoType
	Rect    GoRect

	Init func(d *TestData)

	// The QML provided is run with the initial state above, and
	// then checks are made to ensure the provided state is found.
	QML      string
	QMLLog   string
	QMLValue GoType

	// The function provided is run with the post-QML state above,
	// and then checks are made to ensure the provided state is found.
	Done      func(c *TestData)
	DoneLog   string
	DoneValue GoType
}{
	{
		Summary: "Read a context variable and its fields",
		Value:   GoType{StringValue: "<content>", IntValue: 42},
		QML: `
			Item {
				Component.onCompleted: {
					console.log("String is", value.stringValue)
					console.log("Int is", value.intValue)
					console.log("Any is", value.anyValue)
				}
			}
		`,
		QMLLog: "String is <content>.*Int is 42.*Any is undefined",
	},
	{
		Summary: "Read a nested field via a value (not pointer) in an interface",
		Value:   GoType{AnyValue: struct{ StringValue string }{"<content>"}},
		QML:     `Item { Component.onCompleted: console.log("String is", value.anyValue.stringValue) }`,
		QMLLog:  "String is <content>",
	},
	{
		Summary: "Read a native property",
		QML:     `Item { width: 123 }`,
		Done:    func(c *TestData) { c.Check(c.root.Int("width"), Equals, 123) },
	},
	{
		Summary: "Read object properties",
		QML: `
			Item {
				property bool boolp: true
				property int intp: 1
				property var int64p: 4294967296
				property real float32p: 1.1
				property double float64p: 1.1
				property string stringp: "<content>"
				property var objectp: Rectangle { width: 123 }
				property var nilp: null
			}
		`,
		Done: func(c *TestData) {
			obj := c.root
			c.Check(obj.Bool("boolp"), Equals, true)
			c.Check(obj.Int("intp"), Equals, 1)
			c.Check(obj.Int64("intp"), Equals, int64(1))
			c.Check(obj.Int64("int64p"), Equals, int64(4294967296))
			c.Check(obj.Float64("intp"), Equals, float64(1))
			c.Check(obj.Float64("int64p"), Equals, float64(4294967296))
			c.Check(obj.Float64("float32p"), Equals, float64(1.1))
			c.Check(obj.Float64("float64p"), Equals, float64(1.1))
			c.Check(obj.String("stringp"), Equals, "<content>")
			c.Check(obj.Object("objectp").Int("width"), Equals, 123)
			c.Check(obj.Property("nilp"), Equals, nil)

			c.Check(func() { obj.Bool("intp") }, Panics, `value of property "intp" is not a bool: 1`)
			c.Check(func() { obj.Int("boolp") }, Panics, `value of property "boolp" cannot be represented as an int: true`)
			c.Check(func() { obj.Int64("boolp") }, Panics, `value of property "boolp" cannot be represented as an int64: true`)
			c.Check(func() { obj.Float64("boolp") }, Panics, `value of property "boolp" cannot be represented as a float64: true`)
			c.Check(func() { obj.String("boolp") }, Panics, `value of property "boolp" is not a string: true`)
			c.Check(func() { obj.Object("boolp") }, Panics, `value of property "boolp" is not a QML object: true`)
			c.Check(func() { obj.Property("missing") }, Panics, `object does not have a "missing" property`)
		},
	},
	{
		Summary: "Lowercasing of object properties",
		Init: func(c *TestData) {
			obj := struct{ THE, THEName, Name, N string }{"<a>", "<b>", "<c>", "<d>"}
			c.context.SetVar("obj", &obj)
		},
		QML:    `Item { Component.onCompleted: console.log("Names are", obj.the, obj.theName, obj.name, obj.n) }`,
		QMLLog: "Names are <a> <b> <c> <d>",
	},
	{
		Summary: "No access to private fields",
		Value:   GoType{private: true},
		QML:     `Item { Component.onCompleted: console.log("Private is", value.private); }`,
		QMLLog:  "Private is undefined",
	},
	{
		Summary: "Set a custom property",
		QML: `
			Item {
				property var obj: null

				onObjChanged:     console.log("String is", obj.stringValue)
				onWidthChanged:   console.log("Width is", width)
				onHeightChanged:  console.log("Height is", height)
			}
		`,
		Done: func(c *TestData) {
			value := GoType{StringValue: "<content>"}
			c.root.Set("obj", &value)
			c.root.Set("width", 300)
			c.root.Set("height", 200)
		},
		DoneLog: "String is <content>.*Width is 300.*Height is 200",
	},
	{
		Summary: "Read and set a QUrl property",
		QML:     `import QtWebKit 3.0; WebView {}`,
		Done: func(c *TestData) {
			c.Check(c.root.String("url"), Equals, "")
			url := "http://localhost:54321"
			c.root.Set("url", url)
			c.Check(c.root.String("url"), Equals, url)
		},
	},
	{
		Summary: "Read and set a QColor property",
		QML:     `Text{ color: Qt.rgba(1/16, 1/8, 1/4, 1/2); function hasColor(c) { return Qt.colorEqual(color, c) }}`,
		Done: func(c *TestData) {
			c.Assert(c.root.Color("color"), Equals, color.RGBA{256 / 16, 256 / 8, 256 / 4, 256 / 2})
			c.root.Set("color", color.RGBA{256 / 2, 256 / 4, 256 / 8, 256 / 16})
			c.Assert(c.root.Call("hasColor", color.RGBA{256 / 2, 256 / 4, 256 / 8, 256 / 16}), Equals, true)
		},
	},
	{
		Summary: "Read and set a QColor property from a Go field",
		Init:    func(c *TestData) { c.value.ColorValue = color.RGBA{256 / 16, 256 / 8, 256 / 4, 256 / 2} },
		QML:     `Text{ property var c: value.colorValue; Component.onCompleted: { console.log(value.colorValue); } }`,
		Done: func(c *TestData) {
			c.Assert(c.root.Color("c"), Equals, color.RGBA{256 / 16, 256 / 8, 256 / 4, 256 / 2})
		},
	},
	{
		Summary: "Read a QQmlListProperty property into a Go slice",
		QML: `
			Item {
				states: [
					State { id: on;  name: "on" },
					State { id: off; name: "off" }
				]
			}
		`,
		Done: func(c *TestData) {
			var states []qml.Object
			c.root.List("states").Convert(&states)
			c.Assert(states[0].String("name"), Equals, "on")
			c.Assert(states[1].String("name"), Equals, "off")
			c.Assert(len(states), Equals, 2)
			c.Assert(c.root.Property("states").(*qml.List).Len(), Equals, 2)
		},
	},
	{
		Summary: "Read a QQmlListReference property into a Go slice",
		QML: `
			Item {
				property list<State> mystates: [
					State { id: on;  name: "on" },
					State { id: off; name: "off" }
				]
				Component.onCompleted: value.objectsValue = mystates
			}
		`,
		Done: func(c *TestData) {
			var states []qml.Object
			c.root.List("mystates").Convert(&states)
			c.Assert(states[0].String("name"), Equals, "on")
			c.Assert(states[1].String("name"), Equals, "off")
			c.Assert(len(states), Equals, 2)
			c.Assert(len(c.value.ObjectsValue), Equals, 2)
		},
	},
	{
		Summary: "Read a QVariantList property into a Go slice",
		QML: `
			Item {
				State { id: on;  name: "on" }
				State { id: off; name: "off" }
				property var mystates: [on, off]
			}
		`,
		Done: func(c *TestData) {
			var states []qml.Object
			c.root.List("mystates").Convert(&states)
			c.Assert(states[0].String("name"), Equals, "on")
			c.Assert(states[1].String("name"), Equals, "off")
			c.Assert(len(states), Equals, 2)
		},
	},
	{
		Summary:  "Set a Go slice property",
		QML:      `Item { Component.onCompleted: value.intsValue = [1, 2, 3.5] }`,
		QMLValue: GoType{IntsValue: []int{1, 2, 3}},
	},
	{
		Summary: "Set a Go slice property with objects",
		QML: `
			Item {
				State { id: on;  name: "on" }
				State { id: off; name: "off" }
				Component.onCompleted: value.objectsValue = [on, off]
			}
		`,
		Done: func(c *TestData) {
			c.Assert(c.value.ObjectsValue[0].String("name"), Equals, "on")
			c.Assert(c.value.ObjectsValue[1].String("name"), Equals, "off")
			c.Assert(len(c.value.ObjectsValue), Equals, 2)
		},
	},
	{
		Summary:  "Call a method with a JSON object (issue #48)",
		QML:      `Item { Component.onCompleted: value.setMapValue({a: 1, b: 2}) }`,
		QMLValue: GoType{MapValue: map[string]interface{}{"a": 1, "b": 2}},
	},
	{
		Summary: "Read a map from a QML property",
		QML:     `Item { property var m: {"a": 1, "b": 2} }`,
		Done: func(c *TestData) {
			var m1 map[string]interface{}
			var m2 map[string]int
			m := c.root.Map("m")
			m.Convert(&m1)
			m.Convert(&m2)
			c.Assert(m1, DeepEquals, map[string]interface{}{"a": 1, "b": 2})
			c.Assert(m2, DeepEquals, map[string]int{"a": 1, "b": 2})
			c.Assert(m.Len(), Equals, 2)
		},
	},
	{
		Summary: "Identical values remain identical when possible",
		Init: func(c *TestData) {
			c.context.SetVar("a", c.value)
			c.context.SetVar("b", c.value)
		},
		QML:    `Item { Component.onCompleted: console.log('Identical:', a === b); }`,
		QMLLog: "Identical: true",
	},
	{
		Summary: "Object finding via ObjectByName",
		QML:     `Item { Item { objectName: "subitem"; property string s: "<found>" } }`,
		Done: func(c *TestData) {
			obj := c.root.ObjectByName("subitem")
			c.Check(obj.String("s"), Equals, "<found>")
			c.Check(func() { c.root.ObjectByName("foo") }, Panics, `cannot find descendant with objectName == "foo"`)
		},
	},
	{
		Summary: "Object finding via ObjectByName on GoType",
		QML:     `Item { GoType { objectName: "subitem"; property string s: "<found>" } }`,
		Done: func(c *TestData) {
			obj := c.root.ObjectByName("subitem")
			c.Check(obj.String("s"), Equals, "<found>")
			c.Check(func() { c.root.ObjectByName("foo") }, Panics, `cannot find descendant with objectName == "foo"`)
		},
	},
	{
		Summary: "Register Go type",
		QML:     `GoType { objectName: "test"; Component.onCompleted: console.log("String is", stringValue) }`,
		QMLLog:  "String is <initial>",
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].object.String("objectName"), Equals, "test")
		},
	},
	{
		Summary: "Register Go type with an explicit name",
		QML:     `NamedGoType { objectName: "test"; Component.onCompleted: console.log("String is", stringValue) }`,
		QMLLog:  "String is <initial>",
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].object.String("objectName"), Equals, "test")
		},
	},
	{
		Summary: "Write Go type property",
		QML:     `GoType { stringValue: "<content>"; intValue: 300 }`,
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].StringValue, Equals, "<content>")
			c.Assert(c.createdValue[0].IntValue, Equals, 300)
		},
	},
	{
		Summary: "Write Go type property that has a setter",
		QML:     `GoType { setterStringValue: "<content>" }`,
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].SetterStringValue, Equals, "")
			c.Assert(c.createdValue[0].setterStringValueChanged, Equals, 1)
			c.Assert(c.createdValue[0].setterStringValueSet, Equals, "<content>")
		},
	},
	{
		Summary: "Write Go type property that has a setter and a getter",
		QML: `
			GoType {
				getterStringValue: "<content>"
				Component.onCompleted: console.log("Getter returned", getterStringValue)
			}
		`,
		QMLLog: `Getter returned <content>`,
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].getterStringValue, Equals, "<content>")
			c.Assert(c.createdValue[0].getterStringValueChanged, Equals, 1)
		},
	},
	{
		Summary: "Write an inline object list to a Go type property",
		QML: `
			GoType {
				objectsValue: [State{ name: "on" }, State{ name: "off" }]
				Component.onCompleted: {
					console.log("Length:", objectsValue.length)
					console.log("Name:", objectsValue[0].name)
				}
			}
		`,
		QMLLog: "Length: 2.*Name: on",
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].ObjectsValue[0].String("name"), Equals, "on")
			c.Assert(c.createdValue[0].ObjectsValue[1].String("name"), Equals, "off")
			c.Assert(c.createdValue[0].ObjectsValue, HasLen, 2)
		},
	},
	{
		Summary: "Write an inline object list to a Go type property that has a setter",
		QML:     `GoType { setterObjectsValue: [State{ name: "on" }, State{ name: "off" }] }`,
		Done: func(c *TestData) {
			// Note that the setter is not actually updating the field value, for testing purposes.
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].SetterObjectsValue, IsNil)
			c.Assert(c.createdValue[0].setterObjectsValueChanged, Equals, 2)
			c.Assert(c.createdValue[0].setterObjectsValueSet, HasLen, 1)
			c.Assert(c.createdValue[0].setterObjectsValueSet[0].String("name"), Equals, "off")
		},
	},
	{
		Summary: "Clear an object list in a Go type property",
		QML: `
			GoType {
				objectsValue: [State{ name: "on" }, State{ name: "off" }]
				Component.onCompleted: objectsValue = []
			}
		`,
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].ObjectsValue, HasLen, 0)
		},
	},
	{
		Summary: "Clear an object list in a Go type property that has a setter",
		Value:   GoType{SetterObjectsValue: []qml.Object{nil, nil}},
		QML: `
			GoType {
				objectsValue: [State{ name: "on" }, State{ name: "off" }]
				function clear() { setterObjectsValue = [] }
			}
		`,
		Done: func(c *TestData) {
			// Note that the setter is not actually updating the field value, for testing purposes.
			c.Assert(c.createdValue, HasLen, 1)
			c.createdValue[0].SetterObjectsValue = c.createdValue[0].ObjectsValue

			c.createdValue[0].object.Call("clear")

			c.Assert(c.createdValue[0].SetterObjectsValue, HasLen, 2)
			c.Assert(c.createdValue[0].setterObjectsValueChanged, Equals, 1)
			c.Assert(c.createdValue[0].setterObjectsValueSet, DeepEquals, []qml.Object{})
			c.Assert(&c.createdValue[0].SetterObjectsValue[0], Equals, &c.createdValue[0].ObjectsValue[0])
		},
	},
	{
		Summary: "Access underlying Go value with Interface",
		QML:     `GoType { stringValue: "<content>" }`,
		Done: func(c *TestData) {
			c.Assert(c.root.Interface().(*GoType).StringValue, Equals, "<content>")
			c.Assert(c.context.Interface, Panics, "QML object is not backed by a Go value")
		},
	},
	{
		Summary: "Notification signals on custom Go type",
		QML: `
			GoType {
				id: custom
				stringValue: "<old>"
				onStringValueChanged: if (custom.stringValue != "<newest>") { custom.stringValue = "<newest>" }
				Component.onCompleted: custom.stringValue = "<new>"
			}
		`,
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			c.Assert(c.createdValue[0].StringValue, Equals, "<newest>")
		},
	},
	{
		Summary: "Singleton type registration",
		QML:     `Item { Component.onCompleted: console.log("String is", GoSingleton.stringValue) }`,
		QMLLog:  "String is <initial>",
	},
	{
		Summary: "qml.Changed on unknown value is okay",
		Value:   GoType{StringValue: "<old>"},
		Init: func(c *TestData) {
			value := &GoType{}
			qml.Changed(&value, &value.StringValue)
		},
		QML: `Item{}`,
	},
	{
		Summary: "qml.Changed triggers a QML slot",
		QML: `
			GoType {
				stringValue: "<old>"
				onStringValueChanged: console.log("String is", stringValue)
				onStringAddrValueChanged: console.log("String at addr is", stringAddrValue)
			}
		`,
		QMLLog: "!String is",
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			value := c.createdValue[0]
			s := "<new at addr>"
			value.StringValue = "<new>"
			value.StringAddrValue = &s
			qml.Changed(value, &value.StringValue)
			qml.Changed(value, &value.StringAddrValue)
		},
		DoneLog: "String is <new>.*String at addr is <new at addr>",
	},
	{
		Summary: "qml.Changed must not trigger on the wrong field",
		QML: `
			GoType {
				stringValue: "<old>"
				onStringValueChanged: console.log("String is", stringValue)
			}
		`,
		Done: func(c *TestData) {
			c.Assert(c.createdValue, HasLen, 1)
			value := c.createdValue[0]
			value.StringValue = "<new>"
			qml.Changed(value, &value.IntValue)
		},
		DoneLog: "!String is",
	},
	{
		Summary: "qml.Changed updates bindings",
		Value:   GoType{StringValue: "<old>"},
		QML:     `Item { property string s: "String is " + value.stringValue }`,
		Done: func(c *TestData) {
			c.value.StringValue = "<new>"
			qml.Changed(c.value, &c.value.StringValue)
			c.Check(c.root.String("s"), Equals, "String is <new>")
		},
	},
	{
		Summary:  "Call a Go method without arguments or result",
		Value:    GoType{IntValue: 42},
		QML:      `Item { Component.onCompleted: console.log("Undefined is", value.incrementInt()); }`,
		QMLLog:   "Undefined is undefined",
		QMLValue: GoType{IntValue: 43},
	},
	{
		Summary:  "Call a Go method with one argument and one result",
		Value:    GoType{StringValue: "<old>"},
		QML:      `Item { Component.onCompleted: console.log("String was", value.changeString("<new>")); }`,
		QMLLog:   "String was <old>",
		QMLValue: GoType{StringValue: "<new>"},
	},
	{
		Summary: "Call a Go method with multiple results",
		QML: `
			Item {
				Component.onCompleted: {
					var r = value.mod(42, 4);
					console.log("mod is", r[0], "and err is", r[1]);
				}
			}
		`,
		QMLLog: `mod is 2 and err is undefined`,
	},
	{
		Summary: "Call a Go method that returns an error",
		QML: `
			Item {
				Component.onCompleted: {
					var r = value.mod(0, 0);
					console.log("err is", r[1].error());
				}
			}
		`,
		QMLLog: `err is <division by zero>`,
	},
	{
		Summary: "Call a Go method that recurses back into the GUI thread",
		QML: `
			Item {
				Connections {
					target: value
					onStringValueChanged: console.log("Notification arrived")
				}
				Component.onCompleted: {
					value.notifyStringChanged()
				}
			}
		`,
		QMLLog: "Notification arrived",
	},
	{
		Summary: "Connect a QML signal to a Go method",
		Value:   GoType{StringValue: "<old>"},
		QML: `
			Item {
				id: item
				signal testSignal(string s)
				Component.onCompleted: {
					item.testSignal.connect(value.changeString)
					item.testSignal("<new>")
				}
			}
		`,
		QMLValue: GoType{StringValue: "<new>"},
	},
	{
		Summary: "Call a QML method with no result or parameters from Go",
		QML:     `Item { function f() { console.log("f was called"); } }`,
		Done:    func(c *TestData) { c.Check(c.root.Call("f"), IsNil) },
		DoneLog: "f was called",
	},
	{
		Summary: "Call a QML method with result and parameters from Go",
		QML:     `Item { function add(a, b) { return a+b; } }`,
		Done:    func(c *TestData) { c.Check(c.root.Call("add", 1, 2.1), Equals, float64(3.1)) },
	},
	{
		Summary: "Call a QML method with a custom type",
		Value:   GoType{StringValue: "<content>"},
		QML:     `Item { function log(value) { console.log("String is", value.stringValue) } }`,
		Done:    func(c *TestData) { c.root.Call("log", c.value) },
		DoneLog: "String is <content>",
	},
	{
		Summary: "Call a QML method that returns a QML object",
		QML: `
			Item {
				property var custom: Rectangle { width: 300; }
				function f() { return custom }
			}
		`,
		Done: func(c *TestData) {
			c.Check(c.root.Call("f").(qml.Object).Int("width"), Equals, 300)
		},
	},
	{
		Summary: "Call a QML method that holds a custom type past the return point",
		QML: `
			Item {
				property var held
				function hold(v) { held = v; gc(); gc(); }
				function log()   { console.log("String is", held.stringValue) }
			}`,
		Done: func(c *TestData) {
			value := GoType{StringValue: "<content>"}
			stats := qml.Stats()
			c.root.Call("hold", &value)
			c.Check(qml.Stats().ValuesAlive, Equals, stats.ValuesAlive+1)
			c.root.Call("log")
			c.root.Call("hold", nil)
			c.Check(qml.Stats().ValuesAlive, Equals, stats.ValuesAlive)
		},
		DoneLog: "String is <content>",
	},
	{
		Summary: "Call a non-existent QML method",
		QML:     `Item {}`,
		Done: func(c *TestData) {
			c.Check(func() { c.root.Call("add", 1, 2) }, Panics, `object does not expose a method "add"`)
		},
	},
	{
		Summary: "Ensure URL of provided file is correct by loading a local file",
		Init: func(c *TestData) {
			data, err := base64.StdEncoding.DecodeString("R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==")
			c.Assert(err, IsNil)
			err = ioutil.WriteFile("test.gif", data, 0644)
			c.Check(err, IsNil)
		},
		QML:    `Image { source: "test.gif"; Component.onCompleted: console.log("Ready:", status == Image.Ready) }`,
		QMLLog: "Ready: true",
		Done:   func(c *TestData) { os.Remove("test.gif") },
	},
	{
		Summary: "Create window with non-window root object",
		QML:     `Rectangle { width: 300; height: 200; function inc(x) { return x+1 } }`,
		Done: func(c *TestData) {
			win := c.component.CreateWindow(nil)
			root := win.Root()
			c.Check(root.Int("width"), Equals, 300)
			c.Check(root.Int("height"), Equals, 200)
			c.Check(root.Call("inc", 42.5), Equals, float64(43.5))
			root.Destroy()
		},
	},
	{
		Summary: "Create window with window root object",
		QML: `
			import QtQuick.Window 2.0
			Window { title: "<title>"; width: 300; height: 200 }
		`,
		Done: func(c *TestData) {
			win := c.component.CreateWindow(nil)
			root := win.Root()
			c.Check(root.String("title"), Equals, "<title>")
			c.Check(root.Int("width"), Equals, 300)
			c.Check(root.Int("height"), Equals, 200)
		},
	},
	{
		Summary: "Window is object",
		QML:     `Item {}`,
		Done: func(c *TestData) {
			win := c.component.CreateWindow(nil)
			c.Assert(win.Int("status"), Equals, 1) // Ready
		},
	},
	{
		Summary: "Pass a *Value back into a method",
		QML:     `Rectangle { width: 300; function log(r) { console.log("Width is", r.width) } }`,
		Done:    func(c *TestData) { c.root.Call("log", c.root) },
		DoneLog: "Width is 300",
	},
	{
		Summary: "Create a QML-defined component in Go",
		QML:     `Item { property var comp: Component { Rectangle { width: 300 } } }`,
		Done: func(c *TestData) {
			rect := c.root.Object("comp").Create(nil)
			c.Check(rect.Int("width"), Equals, 300)
			c.Check(func() { c.root.Create(nil) }, Panics, "object is not a component")
			c.Check(func() { c.root.CreateWindow(nil) }, Panics, "object is not a component")
		},
	},
	{
		Summary: "Call a Qt method that has no result",
		QML:     `Item { Component.onDestruction: console.log("item destroyed") }`,
		Done: func(c *TestData) {
			// Create a local instance to avoid double-destroying it.
			root := c.component.Create(nil)
			root.Call("deleteLater")
			time.Sleep(100 * time.Millisecond)
		},
		DoneLog: "item destroyed",
	},
	{
		Summary: "Errors connecting to QML signals",
		QML:     `Item { signal doIt() }`,
		Done: func(c *TestData) {
			c.Check(func() { c.root.On("missing", func() {}) }, Panics, `object does not expose a "missing" signal`)
			c.Check(func() { c.root.On("doIt", func(s string) {}) }, Panics, `signal "doIt" has too few parameters for provided function`)
		},
	},
	{
		Summary: "Connect to a QML signal without parameters",
		QML: `
			Item {
				id: item
				signal doIt()
				function emitDoIt() { item.doIt() }
			}
		`,
		Done: func(c *TestData) {
			itWorks := false
			c.root.On("doIt", func() { itWorks = true })
			c.Check(itWorks, Equals, false)
			c.root.Call("emitDoIt")
			c.Check(itWorks, Equals, true)
		},
	},
	{
		Summary: "Connect to a QML signal with a parameters",
		QML: `
			Item {
				id: item
				signal doIt(string s, int n)
				function emitDoIt() { item.doIt("<arg>", 123) }
			}
		`,
		Done: func(c *TestData) {
			var stack []interface{}
			c.root.On("doIt", func() { stack = append(stack, "A") })
			c.root.On("doIt", func(s string) { stack = append(stack, "B", s) })
			c.root.On("doIt", func(s string, i int) { stack = append(stack, "C", s, i) })
			c.Check(stack, IsNil)
			c.root.Call("emitDoIt")
			c.Check(stack, DeepEquals, []interface{}{"A", "B", "<arg>", "C", "<arg>", 123})
		},
	},
	{
		Summary: "Connect to a QML signal with an object parameter",
		QML:     `import QtWebKit 3.0; WebView{}`,
		Done: func(c *TestData) {
			url := "http://localhost:54321/"
			done := make(chan bool)
			c.root.On("navigationRequested", func(request qml.Object) {
				c.Check(request.String("url"), Equals, url)
				done <- true
			})
			c.root.Set("url", url)
			<-done
		},
	},
	{
		Summary: "Load image from Go provider",
		Init: func(c *TestData) {
			c.engine.AddImageProvider("myprov", func(id string, width, height int) image.Image {
				return image.NewRGBA(image.Rect(0, 0, 200, 100))
			})
		},
		QML: `
			Image {
				source: "image://myprov/myid.png"
				Component.onCompleted: console.log("Size:", width, height)
			}
		`,
		QMLLog: "Size: 200 100",
	},
	{
		Summary: "TypeName",
		QML:     `Item{}`,
		Done:    func(c *TestData) { c.Assert(c.root.TypeName(), Equals, "QQuickItem") },
	},
	{
		Summary: "Custom Go type with painting",
		QML: `
			Rectangle {
				width: 200; height: 200
				color: "black"
				GoRect {
					width: 100; height: 100; x: 50; y: 50
				}
			}
		`,
		Done: func(c *TestData) {
			c.Assert(c.createdRect, HasLen, 0)

			window := c.component.CreateWindow(nil)
			defer window.Destroy()
			window.Show()

			// Qt doesn't hide the Window if we call it too quickly. :-(
			time.Sleep(100 * time.Millisecond)

			c.Assert(c.createdRect, HasLen, 1)
			c.Assert(c.createdRect[0].PaintCount, Equals, 1)

			image := window.Snapshot()
			c.Assert(image.At(25, 25), Equals, color.RGBA{0, 0, 0, 255})
			c.Assert(image.At(100, 100), Equals, color.RGBA{255, 0, 0, 255})
		},
	},
	{
		Summary: "Set a property with the wrong type",
		QML: `
			import QtQuick.Window 2.0
			Window { Rectangle { objectName: "rect" } }
		`,
		Done: func(c *TestData) {
			window := c.component.CreateWindow(nil)
			defer window.Destroy()

			root := window.Root() // It's the window itself in this case
			rect := root.ObjectByName("rect")

			c.Assert(func() { rect.Set("parent", root) }, Panics,
				`cannot set property "parent" with type QQuickItem* to value of QQuickWindow*`)
			c.Assert(func() { rect.Set("parent", 42) }, Panics,
				`cannot set property "parent" with type QQuickItem* to value of int`)
			c.Assert(func() { rect.Set("non_existent", 0) }, Panics,
				`cannot set non-existent property "non_existent" on type QQuickRectangle`)
		},
	},
	{
		Summary: "Register a type converter for a signal parameter",
		QML: `
			Item {
				id: item
				property Item self
				signal testSignal(Item obj)
				function emitSignal() { item.testSignal(item) }
				function getSelf() { return item }
				Component.onCompleted: { self = item }
			}
		`,
		Done: func(c *TestData) {
			type Wrapper struct{ Item qml.Object }
			qml.RegisterConverter(c.root.TypeName(), func(engine *qml.Engine, item qml.Object) interface{} {
				return &Wrapper{item}
			})
			defer qml.RegisterConverter(c.root.TypeName(), nil)

			// Check that it works on signal parameters...
			c.root.On("testSignal", func(wrapped *Wrapper) {
				c.Check(wrapped.Item.Addr(), Equals, c.root.Addr())
				c.Logf("Signal has run.")
			})
			c.root.Call("emitSignal")

			// ... on properties ...
			wrapped, ok := c.root.Property("self").(*Wrapper)
			if c.Check(ok, Equals, true) {
				c.Check(wrapped.Item.Addr(), Equals, c.root.Addr())
			}

			// ... and on results.
			wrapped, ok = c.root.Call("getSelf").(*Wrapper)
			if c.Check(ok, Equals, true) {
				c.Check(wrapped.Item.Addr(), Equals, c.root.Addr())
			}

			// Now unregister and ensure it got disabled.
			qml.RegisterConverter(c.root.TypeName(), nil)
			_, ok = c.root.Property("self").(*qml.Common)
			c.Check(ok, Equals, true)
		},
		DoneLog: "Signal has run.",
	},
	{
		Summary: "References handed out must not be GCd (issue #68)",
		Init: func(c *TestData) {
			type B struct{ S string }
			type A struct{ B *B }
			c.context.SetVar("a", &A{&B{}})
		},
		QML: `Item { function f() { var x = [[],[],[]]; gc(); if (!a.b) console.log("BUG"); } }`,
		Done: func(c *TestData) {
			for i := 0; i < 100; i++ {
				c.root.Call("f")
			}
		},
		DoneLog: "!BUG",
	},
}

var tablef = flag.String("tablef", "", "if provided, TestTable only runs tests with a summary matching the regexp")

func (s *S) TestTable(c *C) {
	var testData TestData

	types := []qml.TypeSpec{{
		Init: func(v *GoType, obj qml.Object) {
			v.object = obj
			v.StringValue = "<initial>"
			testData.createdValue = append(testData.createdValue, v)
		},
	}, {
		Name: "NamedGoType",
		Init: func(v *GoType, obj qml.Object) {
			v.object = obj
			v.StringValue = "<initial>"
			testData.createdValue = append(testData.createdValue, v)
		},
	}, {
		Name: "GoSingleton",
		Init: func(v *GoType, obj qml.Object) {
			v.object = obj
			v.StringValue = "<initial>"
			testData.createdSingleton = append(testData.createdSingleton, v)
		},
		Singleton: true,
	}, {
		Init: func(v *GoRect, obj qml.Object) {
			testData.createdRect = append(testData.createdRect, v)
		},
	}}

	qml.RegisterTypes("GoTypes", 4, 2, types)

	filter := regexp.MustCompile("")
	if tablef != nil {
		filter = regexp.MustCompile(*tablef)
	}

	for i := range tests {
		s.TearDownTest(c)
		t := &tests[i]
		header := fmt.Sprintf("----- Running table test %d: %s -----", i, t.Summary)
		if !filter.MatchString(header) {
			continue
		}
		c.Log(header)
		s.SetUpTest(c)

		value := t.Value
		s.context.SetVar("value", &value)

		testData = TestData{
			C:       c,
			value:   &value,
			engine:  s.engine,
			context: s.context,
		}

		if t.Init != nil {
			t.Init(&testData)
			if c.Failed() {
				c.FailNow()
			}
		}

		component, err := s.engine.LoadString("file.qml", "import QtQuick 2.0\nimport GoTypes 4.2\n"+strings.TrimSpace(t.QML))
		c.Assert(err, IsNil)

		logMark := c.GetTestLog()

		// The component instance is destroyed before the loop ends below,
		// but do a defer to ensure it will be destroyed if the test fails.
		root := component.Create(nil)
		defer root.Destroy()

		testData.component = component
		testData.root = root

		if t.QMLLog != "" {
			logged := c.GetTestLog()[len(logMark):]
			if t.QMLLog[0] == '!' {
				c.Check(logged, Not(Matches), "(?s).*"+t.QMLLog[1:]+".*")
			} else {
				c.Check(logged, Matches, "(?s).*"+t.QMLLog+".*")
			}
		}

		if !reflect.DeepEqual(t.QMLValue, GoType{}) {
			c.Check(value.StringValue, Equals, t.QMLValue.StringValue)
			c.Check(value.StringAddrValue, Equals, t.QMLValue.StringAddrValue)
			c.Check(value.BoolValue, Equals, t.QMLValue.BoolValue)
			c.Check(value.IntValue, Equals, t.QMLValue.IntValue)
			c.Check(value.Int64Value, Equals, t.QMLValue.Int64Value)
			c.Check(value.Int32Value, Equals, t.QMLValue.Int32Value)
			c.Check(value.Float64Value, Equals, t.QMLValue.Float64Value)
			c.Check(value.Float32Value, Equals, t.QMLValue.Float32Value)
			c.Check(value.AnyValue, Equals, t.QMLValue.AnyValue)
			c.Check(value.IntsValue, DeepEquals, t.QMLValue.IntsValue)
			c.Check(value.MapValue, DeepEquals, t.QMLValue.MapValue)
		}

		if !c.Failed() {
			logMark := c.GetTestLog()

			if t.Done != nil {
				t.Done(&testData)
			}

			if t.DoneLog != "" {
				logged := c.GetTestLog()[len(logMark):]
				if t.DoneLog[0] == '!' {
					c.Check(logged, Not(Matches), "(?s).*"+t.DoneLog[1:]+".*")
				} else {
					c.Check(logged, Matches, "(?s).*"+t.DoneLog+".*")
				}
			}

			if !reflect.DeepEqual(t.DoneValue, GoType{}) {
				c.Check(value.StringValue, Equals, t.DoneValue.StringValue)
				c.Check(value.StringAddrValue, Equals, t.DoneValue.StringAddrValue)
				c.Check(value.BoolValue, Equals, t.DoneValue.BoolValue)
				c.Check(value.IntValue, Equals, t.DoneValue.IntValue)
				c.Check(value.Int64Value, Equals, t.DoneValue.Int64Value)
				c.Check(value.Int32Value, Equals, t.DoneValue.Int32Value)
				c.Check(value.Float64Value, Equals, t.DoneValue.Float64Value)
				c.Check(value.Float32Value, Equals, t.DoneValue.Float32Value)
				c.Check(value.AnyValue, Equals, t.DoneValue.AnyValue)
				c.Check(value.IntsValue, DeepEquals, t.DoneValue.IntsValue)
				c.Check(value.MapValue, DeepEquals, t.DoneValue.MapValue)
			}
		}

		root.Destroy()

		if c.Failed() {
			c.FailNow() // So relevant logs are at the bottom.
		}
	}
}
