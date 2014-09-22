package main

import (
	"fmt"
	"os"

	"gopkg.in/qml.v1"
)

func ErrorWindow(err error) {
	engine := qml.NewEngine()
	component, e := engine.LoadString("local", qmlErr)
	if e != nil {
		fmt.Println("err:", err)
		os.Exit(1)
	}

	win := component.CreateWindow(nil)
	win.Root().ObjectByName("label").Set("text", err.Error())
	win.Show()
	win.Wait()
}

const qmlErr = `
import QtQuick 2.0; import QtQuick.Controls 1.0;
ApplicationWindow {
	width: 600; height: 150;
	flags: Qt.CustomizeWindowHint | Qt.WindowTitleHint | Qt.WindowCloseButtonHint
	title: "Error"
	Text {
		x: parent.width / 2 - this.width / 2;
		y: parent.height / 2 - this.height / 2;
		objectName: "label";
	}
}
`
