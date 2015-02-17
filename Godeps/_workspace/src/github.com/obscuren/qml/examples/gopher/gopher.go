package main

import (
	"fmt"
	"gopkg.in/qml.v1"
	"gopkg.in/qml.v1/gl/2.0"
	"os"
)

var filename = "gopher.qml"

func main() {
	if len(os.Args) == 2 {
		filename = os.Args[1]
	}
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	engine := qml.NewEngine()

	model, err := Read("model/gopher.obj")
	if err != nil {
		return err
	}

	qml.RegisterTypes("GoExtensions", 1, 0, []qml.TypeSpec{{
		Init: func(g *Gopher, obj qml.Object) {
			g.Object = obj
			g.model = model
		},
	}})

	component, err := engine.LoadFile(filename)
	if err != nil {
		return err
	}

	win := component.CreateWindow(nil)
	win.Set("x", 560)
	win.Set("y", 320)
	win.Show()
	win.Wait()
	return nil
}

type Gopher struct {
	qml.Object

	model map[string]*Object

	Rotation int
}

func (r *Gopher) SetRotation(rotation int) {
	r.Rotation = rotation
	r.Call("update")
}

func (r *Gopher) Paint(p *qml.Painter) {
	gl := GL.API(p)

	width := float32(r.Int("width"))

	gl.Enable(GL.BLEND)
	gl.BlendFunc(GL.SRC_ALPHA, GL.ONE_MINUS_SRC_ALPHA)

	gl.ShadeModel(GL.SMOOTH)
	gl.Enable(GL.DEPTH_TEST)
	gl.DepthMask(true)
	gl.Enable(GL.NORMALIZE)

	gl.Clear(GL.DEPTH_BUFFER_BIT)

	gl.Scalef(width/3, width/3, width/3)

	lka := []float32{0.3, 0.3, 0.3, 1.0}
	lkd := []float32{1.0, 1.0, 1.0, 0.0}
	lks := []float32{1.0, 1.0, 1.0, 1.0}
	lpos := []float32{-2, 6, 3, 1.0}

	gl.Enable(GL.LIGHTING)
	gl.Lightfv(GL.LIGHT0, GL.AMBIENT, lka)
	gl.Lightfv(GL.LIGHT0, GL.DIFFUSE, lkd)
	gl.Lightfv(GL.LIGHT0, GL.SPECULAR, lks)
	gl.Lightfv(GL.LIGHT0, GL.POSITION, lpos)
	gl.Enable(GL.LIGHT0)

	gl.EnableClientState(GL.NORMAL_ARRAY)
	gl.EnableClientState(GL.VERTEX_ARRAY)

	gl.Translatef(1.5, 1.5, 0)
	gl.Rotatef(-90, 0, 0, 1)
	gl.Rotatef(float32(90+((36000+r.Rotation)%360)), 1, 0, 0)

	gl.Disable(GL.COLOR_MATERIAL)

	for _, obj := range r.model {
		for _, group := range obj.Groups {
			gl.Materialfv(GL.FRONT, GL.AMBIENT, group.Material.Ambient)
			gl.Materialfv(GL.FRONT, GL.DIFFUSE, group.Material.Diffuse)
			gl.Materialfv(GL.FRONT, GL.SPECULAR, group.Material.Specular)
			gl.Materialf(GL.FRONT, GL.SHININESS, group.Material.Shininess)
			gl.VertexPointer(3, GL.FLOAT, 0, group.Vertexes)
			gl.NormalPointer(GL.FLOAT, 0, group.Normals)
			gl.DrawArrays(GL.TRIANGLES, 0, len(group.Vertexes)/3)
		}
	}

	gl.Enable(GL.COLOR_MATERIAL)

	gl.DisableClientState(GL.NORMAL_ARRAY)
	gl.DisableClientState(GL.VERTEX_ARRAY)
}
