package main

import (
	"fmt"
	"os"

	"gopkg.in/qml.v1"
)

func main() {
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type GoType struct {
	Text string
}

func (v *GoType) SetText(text string) {
	fmt.Println("Text changing to:", text)
	v.Text = text
}

type GoSingleton struct {
	Event string
}

func run() error {
	qml.RegisterTypes("GoExtensions", 1, 0, []qml.TypeSpec{{
		Init: func(v *GoType, obj qml.Object) {},
	}, {
		Init: func(v *GoSingleton, obj qml.Object) { v.Event = "birthday" },

		Singleton: true,
	}})

	engine := qml.NewEngine()
	component, err := engine.LoadFile("customtype.qml")
	if err != nil {
		return err
	}

	value := component.Create(nil)
	fmt.Println("Text is:", value.Interface().(*GoType).Text)
	return nil
}
