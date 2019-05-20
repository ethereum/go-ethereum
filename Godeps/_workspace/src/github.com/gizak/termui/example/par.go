// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// +build ignore

package main

import "github.com/gizak/termui"

func main() {
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()

	termui.UseTheme("helloworld")

	par0 := termui.NewPar("Borderless Text")
	par0.Height = 1
	par0.Width = 20
	par0.Y = 1
	par0.HasBorder = false

	par1 := termui.NewPar("你好，世界。")
	par1.Height = 3
	par1.Width = 17
	par1.X = 20
	par1.Border.Label = "标签"

	par2 := termui.NewPar("Simple text\nwith label. It can be multilined with \\n or break automatically")
	par2.Height = 5
	par2.Width = 37
	par2.Y = 4
	par2.Border.Label = "Multiline"
	par2.Border.FgColor = termui.ColorYellow

	par3 := termui.NewPar("Long text with label and it is auto trimmed.")
	par3.Height = 3
	par3.Width = 37
	par3.Y = 9
	par3.Border.Label = "Auto Trim"

	termui.Render(par0, par1, par2, par3)

	<-termui.EventCh()
}
