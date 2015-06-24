// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// +build ignore

package main

import (
	"math"

	"github.com/gizak/termui"
)

func main() {
	err := termui.Init()
	if err != nil {
		panic(err)
	}
	defer termui.Close()

	termui.UseTheme("helloworld")

	sinps := (func() []float64 {
		n := 220
		ps := make([]float64, n)
		for i := range ps {
			ps[i] = 1 + math.Sin(float64(i)/5)
		}
		return ps
	})()

	lc0 := termui.NewLineChart()
	lc0.Border.Label = "braille-mode Line Chart"
	lc0.Data = sinps
	lc0.Width = 50
	lc0.Height = 12
	lc0.X = 0
	lc0.Y = 0
	lc0.AxesColor = termui.ColorWhite
	lc0.LineColor = termui.ColorGreen | termui.AttrBold

	lc1 := termui.NewLineChart()
	lc1.Border.Label = "dot-mode Line Chart"
	lc1.Mode = "dot"
	lc1.Data = sinps
	lc1.Width = 26
	lc1.Height = 12
	lc1.X = 51
	lc1.DotStyle = '+'
	lc1.AxesColor = termui.ColorWhite
	lc1.LineColor = termui.ColorYellow | termui.AttrBold

	lc2 := termui.NewLineChart()
	lc2.Border.Label = "dot-mode Line Chart"
	lc2.Mode = "dot"
	lc2.Data = sinps[4:]
	lc2.Width = 77
	lc2.Height = 16
	lc2.X = 0
	lc2.Y = 12
	lc2.AxesColor = termui.ColorWhite
	lc2.LineColor = termui.ColorCyan | termui.AttrBold

	termui.Render(lc0, lc1, lc2)

	<-termui.EventCh()
}
