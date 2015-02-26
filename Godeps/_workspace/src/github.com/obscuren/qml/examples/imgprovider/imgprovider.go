package main

import (
	"fmt"
	"gopkg.in/qml.v1"
	"image"
	"image/png"
	"os"
)

func main() {
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	engine := qml.NewEngine()
	engine.AddImageProvider("pwd", func(id string, width, height int) image.Image {
		f, err := os.Open(id)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		image, err := png.Decode(f)
		if err != nil {
			panic(err)
		}
		return image
	})

	component, err := engine.LoadFile("imgprovider.qml")
	if err != nil {
		return err
	}

	win := component.CreateWindow(nil)
	win.Show()
	win.Wait()

	return nil
}
