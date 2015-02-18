package main

import (
	"fmt"
	"gopkg.in/qml.v1"
	"image/png"
	"os"
)

const webview = `
import QtQuick 2.0
import QtWebKit 3.0

WebView {
    width: 1024
    height: 768
}
`

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <url> <png path>\n", os.Args[0])
		os.Exit(1)
	}
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	engine := qml.NewEngine()
	component, err := engine.LoadString("webview.qml", webview)
	if err != nil {
		return err
	}
	ctrl := &Control{
		done: make(chan error),
		win:  component.CreateWindow(nil),
	}
	engine.Context().SetVar("ctrl", ctrl)
	root := ctrl.win.Root()
	root.On("loadingChanged", ctrl.Snapshot)
	root.Set("url", os.Args[1])
	ctrl.win.Show()
	return <-ctrl.done
}

type Control struct {
	win  *qml.Window
	done chan error
}

func (ctrl *Control) Snapshot(request qml.Object) {
	if request.Int("status") != 2 {
		return
	}
	f, err := os.Create(os.Args[2])
	if err != nil {
		ctrl.done <- err
		return
	}
	defer f.Close()
	img := ctrl.win.Snapshot()
	err = png.Encode(f, img)
	if err != nil {
		os.Remove(os.Args[2])
	}
	ctrl.done <- err
}
