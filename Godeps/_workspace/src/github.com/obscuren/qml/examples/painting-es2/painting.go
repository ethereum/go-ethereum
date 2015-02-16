package main

import (
	"fmt"
	"os"

	"gopkg.in/qml.v1"
	"gopkg.in/qml.v1/gl/es2"
)

func main() {
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	qml.RegisterTypes("GoExtensions", 1, 0, []qml.TypeSpec{{
		Init: func(r *GoRect, obj qml.Object) { r.Object = obj },
	}})

	engine := qml.NewEngine()
	component, err := engine.LoadFile("painting.qml")
	if err != nil {
		return err
	}

	win := component.CreateWindow(nil)
	win.Show()
	win.Wait()
	return nil
}

type GoRect struct {
	qml.Object
}

var vertexShader = `
#version 120

attribute vec2 position;

void main()
{
    gl_Position = vec4(position.x, position.y, 0.0, 1.0);
}
`

var fragmentShader = `
#version 120

void main()
{
    gl_FragColor = vec4(1.0, 1.0, 1.0, 0.8);
}
`

func (r *GoRect) Paint(p *qml.Painter) {
	gl := GL.API(p)

	vertices := []float32{
		-1, -1,
		+1, -1,
		+1, +1,
		-1, +1,
	}

	indices := []uint8{
		0, 1, 2, // first triangle
		2, 3, 0, // second triangle
	}

	buf := gl.GenBuffers(2)
	gl.BindBuffer(GL.ARRAY_BUFFER, buf[0])
	gl.BufferData(GL.ARRAY_BUFFER, 0, vertices, GL.STATIC_DRAW)
	gl.BindBuffer(GL.ELEMENT_ARRAY_BUFFER, buf[1])
	gl.BufferData(GL.ELEMENT_ARRAY_BUFFER, 0, indices, GL.STATIC_DRAW)

	vshader := gl.CreateShader(GL.VERTEX_SHADER);
	gl.ShaderSource(vshader, vertexShader)
	gl.CompileShader(vshader)

	var status [1]int32
	gl.GetShaderiv(vshader, GL.COMPILE_STATUS, status[:])
	if status[0] == 0 {
		log := gl.GetShaderInfoLog(vshader)
		panic("vertex shader compilation failed: " + string(log))
	}

	fshader := gl.CreateShader(GL.FRAGMENT_SHADER)
	gl.ShaderSource(fshader, fragmentShader)
	gl.CompileShader(fshader)

	gl.GetShaderiv(fshader, GL.COMPILE_STATUS, status[:])
	if status[0] == 0 {
		log := gl.GetShaderInfoLog(fshader)
		panic("fragment shader compilation failed: " + string(log))
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vshader)
	gl.AttachShader(program, fshader)
	gl.LinkProgram(program)
	gl.UseProgram(program)

	position := gl.GetAttribLocation(program, "position")
	gl.VertexAttribPointer(position, 2, GL.FLOAT, false, 0, 0)
	gl.EnableVertexAttribArray(position)

	gl.Enable(GL.BLEND)
	gl.BlendFunc(GL.SRC_ALPHA, GL.ONE_MINUS_SRC_ALPHA)

	gl.DrawElements(GL.TRIANGLES, 6, GL.UNSIGNED_BYTE, nil)
}
