// Package qml offers graphical QML application support for the Go language.
//
// Attention
//
// This package is in an alpha stage, and still in heavy development. APIs may
// change, and things may break.
//
// At this time contributors and developers that are interested in tracking the
// development closely are encouraged to use it. If you'd prefer a more stable
// release, please hold on a bit and subscribe to the mailing list for news. It's
// in a pretty good state, so it shall not take too long.
//
// See http://github.com/go-qml/qml for details.
//
//
// Introduction
//
// The qml package enables Go programs to display and manipulate graphical content
// using Qt's QML framework. QML uses a declarative language to express structure
// and style, and supports JavaScript for in-place manipulation of the described
// content. When using the Go qml package, such QML content can also interact with
// Go values, making use of its exported fields and methods, and even explicitly
// creating new instances of registered Go types.
//
// A simple Go application that integrates with QML may perform the following steps
// for offering a graphical interface:
//
//   * Call qml.Run from function main providing a function with the logic below
//   * Create an engine for loading and running QML content (see NewEngine)
//   * Make Go values and types available to QML (see Context.SetVar and RegisterType)
//   * Load QML content (see Engine.LoadString and Engine.LoadFile)
//   * Create a new window for the content (see Component.CreateWindow)
//   * Show the window and wait for it to be closed (see Window.Show and Window.Wait)
//
// Some of these topics are covered below, and may also be observed in practice
// in the following examples:
//
//   https://github.com/go-qml/qml/tree/v1/examples
//
//
// Simple example
//
// The following logic demonstrates loading a QML file into a window:
//
//    func main() {
//            err := qml.Run(run)
//            ...
//    }
//
//    func run() error {
//            engine := qml.NewEngine()
//            component, err := engine.LoadFile("file.qml")
//            if err != nil {
//                    return err
//            }
//            win := component.CreateWindow(nil)
//            win.Show()
//            win.Wait()
//            return nil
//    }
//
// Handling QML objects in Go
//
// Any QML object may be manipulated by Go via the Object interface. That
// interface is implemented both by dynamic QML values obtained from a running
// engine, and by Go types in the qml package that represent QML values, such as
// Window, Context, and Engine.
//
// For example, the following logic creates a window and prints its width
// whenever it's made visible:
//
//    win := component.CreateWindow(nil)
//    win.On("visibleChanged", func(visible bool) {
//            if (visible) {
//                    fmt.Println("Width:", win.Int("width"))
//            }
//    })
//
// Information about the methods, properties, and signals that are available for QML
// objects may be obtained in the Qt documentation. As a reference, the "visibleChanged"
// signal and the "width" property used in the example above are described at:
//
//    http://qt-project.org/doc/qt-5.0/qtgui/qwindow.html
//
// When in doubt about what type is being manipulated, the Object.TypeName method
// provides the type name of the underlying value.
//
//
// Publishing Go values to QML
//
// The simplest way of making a Go value available to QML code is setting it
// as a variable of the engine's root context, as in:
//
//    context := engine.Context()
//    context.SetVar("person", &Person{Name: "Ale"})
//
// This logic would enable the following QML code to successfully run:
//
//    import QtQuick 2.0
//    Item {
//        Component.onCompleted: console.log("Name is", person.name)
//    }
//
//
// Publishing Go types to QML
//
// While registering an individual Go value as described above is a quick way to get
// started, it is also fairly limited. For more flexibility, a Go type may be
// registered so that QML code can natively create new instances in an arbitrary
// position of the structure. This may be achieved via the RegisterType function, as
// the following example demonstrates:
//
//    qml.RegisterTypes("GoExtensions", 1, 0, []qml.TypeSpec{{
//            Init: func(p *Person, obj qml.Object) { p.Name = "<none>" },
//    }})
//
// With this logic in place, QML code can create new instances of Person by itself:     
//
//    import QtQuick 2.0
//    import GoExtensions 1.0
//    Item{
//        Person {
//            id: person
//            name: "Ale"
//        }
//        Component.onCompleted: console.log("Name is", person.name)
//    }
//
//
// Lowercasing of names
// 
// Independently from the mechanism used to publish a Go value to QML code, its methods
// and fields are available to QML logic as methods and properties of the
// respective QML object representing it. As required by QML, though, the Go
// method and field names are lowercased according to the following scheme when
// being accesed from QML:
//
//    value.Name      => value.name
//    value.UPPERName => value.upperName
//    value.UPPER     => value.upper
//
//
// Setters and getters
//
// While QML code can directly read and write exported fields of Go values, as described
// above, a Go type can also intercept writes to specific fields by declaring a setter
// method according to common Go conventions. This is often useful for updating the
// internal state or the visible content of a Go-defined type.
//
// For example:
//
//    type Person struct {
//            Name string
//    }
//
//    func (p *Person) SetName(name string) {
//            fmt.Println("Old name is", p.Name)
//            p.Name = name
//            fmt.Println("New name is", p.Name)
//    }
//
// In the example above, whenever QML code attempts to update the Person.Name field
// via any means (direct assignment, object declarations, etc) the SetName method
// is invoked with the provided value instead.
//
// A setter method may also be used in conjunction with a getter method rather
// than a real type field. A method is only considered a getter in the presence
// of the respective setter, and according to common Go conventions it must not
// have the Get prefix.
//
// Inside QML logic, the getter and setter pair is seen as a single object property.
//
//
// Painting
//
// Custom types implemented in Go may have displayable content by defining
// a Paint method such as:
//
//    func (p *Person) Paint(painter *qml.Painter) {
//            // ... OpenGL calls with the gopkg.in/qml.v1/gl/<VERSION> package ...
//    }
//
// A simple example is available at:
//
//   https://github.com/go-qml/qml/tree/v1/examples/painting
//
//
// Packing resources into the Go qml binary
//
// Resource files (qml code, images, etc) may be packed into the Go qml application
// binary to simplify its handling and distribution. This is done with the genqrc tool:
//
//   http://gopkg.in/qml.v1/cmd/genqrc#usage
//
// The following blog post provides more details:
//
//   http://blog.labix.org/2014/09/26/packing-resources-into-go-qml-binaries
//
package qml
