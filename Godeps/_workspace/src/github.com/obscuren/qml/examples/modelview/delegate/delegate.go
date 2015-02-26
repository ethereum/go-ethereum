package main

import (
	"fmt"
	"gopkg.in/qml.v1"
	"image/color"
	"math/rand"
	"os"
	"time"
)

func main() {
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	engine := qml.NewEngine()
	colors := &Colors{}
	engine.Context().SetVar("colors", colors)
	component, err := engine.LoadFile("delegate.qml")
	if err != nil {
		return err
	}
	window := component.CreateWindow(nil)
	window.Show()
	go func() {
		n := func() uint8 { return uint8(rand.Intn(256)) }
		for i := 0; i < 100; i++ {
			colors.Add(color.RGBA{n(), n(), n(), 0xff})
			time.Sleep(1 * time.Second)
		}
	}()
	window.Wait()
	return nil
}

type Colors struct {
	list []color.RGBA
	Len  int
}

func (colors *Colors) Add(c color.RGBA) {
	colors.list = append(colors.list, c)
	colors.Len = len(colors.list)
	qml.Changed(colors, &colors.Len)
}

func (colors *Colors) Color(index int) color.RGBA {
	return colors.list[index]
}
