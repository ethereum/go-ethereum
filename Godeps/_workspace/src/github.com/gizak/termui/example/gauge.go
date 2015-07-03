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

	g0 := termui.NewGauge()
	g0.Percent = 40
	g0.Width = 50
	g0.Height = 3
	g0.Border.Label = "Slim Gauge"
	g0.BarColor = termui.ColorRed
	g0.Border.FgColor = termui.ColorWhite
	g0.Border.LabelFgColor = termui.ColorCyan

	g2 := termui.NewGauge()
	g2.Percent = 60
	g2.Width = 50
	g2.Height = 3
	g2.PercentColor = termui.ColorBlue
	g2.Y = 3
	g2.Border.Label = "Slim Gauge"
	g2.BarColor = termui.ColorYellow
	g2.Border.FgColor = termui.ColorWhite

	g1 := termui.NewGauge()
	g1.Percent = 30
	g1.Width = 50
	g1.Height = 5
	g1.Y = 6
	g1.Border.Label = "Big Gauge"
	g1.PercentColor = termui.ColorYellow
	g1.BarColor = termui.ColorGreen
	g1.Border.FgColor = termui.ColorWhite
	g1.Border.LabelFgColor = termui.ColorMagenta

	g3 := termui.NewGauge()
	g3.Percent = 50
	g3.Width = 50
	g3.Height = 3
	g3.Y = 11
	g3.Border.Label = "Gauge with custom label"
	g3.Label = "{{percent}}% (100MBs free)"
	g3.LabelAlign = termui.AlignRight

	termui.Render(g0, g1, g2, g3)

	<-termui.EventCh()
}
