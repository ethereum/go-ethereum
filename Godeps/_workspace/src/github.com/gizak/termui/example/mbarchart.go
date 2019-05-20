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

	bc := termui.NewMBarChart()
	math := []int{90, 85, 90, 80}
	english := []int{70, 85, 75, 60}
	science := []int{75, 60, 80, 85}
	compsci := []int{100, 100, 100, 100}
	bc.Data[0] = math
	bc.Data[1] = english
	bc.Data[2] = science
	bc.Data[3] = compsci
	studentsName := []string{"Ken", "Rob", "Dennis", "Linus"}
	bc.Border.Label = "Student's Marks X-Axis=Name Y-Axis=Marks[Math,English,Science,ComputerScience] in %"
	bc.Width = 100
	bc.Height = 50
	bc.Y = 10
	bc.BarWidth = 10
	bc.DataLabels = studentsName
	bc.ShowScale = true //Show y_axis scale value (min and max)
	bc.SetMax(400)

	bc.TextColor = termui.ColorGreen    //this is color for label (x-axis)
	bc.BarColor[3] = termui.ColorGreen  //BarColor for computerscience
	bc.BarColor[1] = termui.ColorYellow //Bar Color for english
	bc.NumColor[3] = termui.ColorRed    // Num color for computerscience
	bc.NumColor[1] = termui.ColorRed    // num color for english

	//Other colors are automatically populated, btw All the students seems do well in computerscience. :p

	termui.Render(bc)

	<-termui.EventCh()
}
