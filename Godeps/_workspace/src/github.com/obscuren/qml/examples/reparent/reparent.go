package main

import (
	"fmt"
	"gopkg.in/qml.v1"
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

	base, err := engine.LoadFile("base.qml")
	if err != nil {
		return err
	}
	rect, err := engine.LoadFile("rect.qml")
	if err != nil {
		return err
	}

	win := base.CreateWindow(nil)
	obj := rect.Create(nil)

	obj.Set("parent", win.Root())

	win.Show()
	win.Wait()

	return nil
}
