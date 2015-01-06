/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
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
