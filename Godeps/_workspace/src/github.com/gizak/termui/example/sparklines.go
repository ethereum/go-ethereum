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

	data := []int{4, 2, 1, 6, 3, 9, 1, 4, 2, 15, 14, 9, 8, 6, 10, 13, 15, 12, 10, 5, 3, 6, 1, 7, 10, 10, 14, 13, 6}
	spl0 := termui.NewSparkline()
	spl0.Data = data[3:]
	spl0.Title = "Sparkline 0"
	spl0.LineColor = termui.ColorGreen

	// single
	spls0 := termui.NewSparklines(spl0)
	spls0.Height = 2
	spls0.Width = 20
	spls0.HasBorder = false

	spl1 := termui.NewSparkline()
	spl1.Data = data
	spl1.Title = "Sparkline 1"
	spl1.LineColor = termui.ColorRed

	spl2 := termui.NewSparkline()
	spl2.Data = data[5:]
	spl2.Title = "Sparkline 2"
	spl2.LineColor = termui.ColorMagenta

	// group
	spls1 := termui.NewSparklines(spl0, spl1, spl2)
	spls1.Height = 8
	spls1.Width = 20
	spls1.Y = 3
	spls1.Border.Label = "Group Sparklines"

	spl3 := termui.NewSparkline()
	spl3.Data = data
	spl3.Title = "Enlarged Sparkline"
	spl3.Height = 8
	spl3.LineColor = termui.ColorYellow

	spls2 := termui.NewSparklines(spl3)
	spls2.Height = 11
	spls2.Width = 30
	spls2.Border.FgColor = termui.ColorCyan
	spls2.X = 21
	spls2.Border.Label = "Tweeked Sparkline"

	termui.Render(spls0, spls1, spls2)

	<-termui.EventCh()
}
