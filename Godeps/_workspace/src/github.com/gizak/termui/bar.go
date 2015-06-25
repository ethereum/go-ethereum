// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import "fmt"

// BarChart creates multiple bars in a widget:
/*
   bc := termui.NewBarChart()
   data := []int{3, 2, 5, 3, 9, 5}
   bclabels := []string{"S0", "S1", "S2", "S3", "S4", "S5"}
   bc.Border.Label = "Bar Chart"
   bc.Data = data
   bc.Width = 26
   bc.Height = 10
   bc.DataLabels = bclabels
   bc.TextColor = termui.ColorGreen
   bc.BarColor = termui.ColorRed
   bc.NumColor = termui.ColorYellow
*/
type BarChart struct {
	Block
	BarColor   Attribute
	TextColor  Attribute
	NumColor   Attribute
	Data       []int
	DataLabels []string
	BarWidth   int
	BarGap     int
	labels     [][]rune
	dataNum    [][]rune
	numBar     int
	scale      float64
	max        int
}

// NewBarChart returns a new *BarChart with current theme.
func NewBarChart() *BarChart {
	bc := &BarChart{Block: *NewBlock()}
	bc.BarColor = theme.BarChartBar
	bc.NumColor = theme.BarChartNum
	bc.TextColor = theme.BarChartText
	bc.BarGap = 1
	bc.BarWidth = 3
	return bc
}

func (bc *BarChart) layout() {
	bc.numBar = bc.innerWidth / (bc.BarGap + bc.BarWidth)
	bc.labels = make([][]rune, bc.numBar)
	bc.dataNum = make([][]rune, len(bc.Data))

	for i := 0; i < bc.numBar && i < len(bc.DataLabels) && i < len(bc.Data); i++ {
		bc.labels[i] = trimStr2Runes(bc.DataLabels[i], bc.BarWidth)
		n := bc.Data[i]
		s := fmt.Sprint(n)
		bc.dataNum[i] = trimStr2Runes(s, bc.BarWidth)
	}

	//bc.max = bc.Data[0] //  what if Data is nil? Sometimes when bar graph is nill it produces panic with panic: runtime error: index out of range
	// Asign a negative value to get maxvalue auto-populates
	if bc.max == 0 {
		bc.max = -1
	}
	for i := 0; i < len(bc.Data); i++ {
		if bc.max < bc.Data[i] {
			bc.max = bc.Data[i]
		}
	}
	bc.scale = float64(bc.max) / float64(bc.innerHeight-1)
}

func (bc *BarChart) SetMax(max int) {

	if max > 0 {
		bc.max = max
	}
}

// Buffer implements Bufferer interface.
func (bc *BarChart) Buffer() []Point {
	ps := bc.Block.Buffer()
	bc.layout()

	for i := 0; i < bc.numBar && i < len(bc.Data) && i < len(bc.DataLabels); i++ {
		h := int(float64(bc.Data[i]) / bc.scale)
		oftX := i * (bc.BarWidth + bc.BarGap)
		// plot bar
		for j := 0; j < bc.BarWidth; j++ {
			for k := 0; k < h; k++ {
				p := Point{}
				p.Ch = ' '
				p.Bg = bc.BarColor
				if bc.BarColor == ColorDefault { // when color is default, space char treated as transparent!
					p.Bg |= AttrReverse
				}
				p.X = bc.innerX + i*(bc.BarWidth+bc.BarGap) + j
				p.Y = bc.innerY + bc.innerHeight - 2 - k
				ps = append(ps, p)
			}
		}
		// plot text
		for j, k := 0, 0; j < len(bc.labels[i]); j++ {
			w := charWidth(bc.labels[i][j])
			p := Point{}
			p.Ch = bc.labels[i][j]
			p.Bg = bc.BgColor
			p.Fg = bc.TextColor
			p.Y = bc.innerY + bc.innerHeight - 1
			p.X = bc.innerX + oftX + k
			ps = append(ps, p)
			k += w
		}
		// plot num
		for j := 0; j < len(bc.dataNum[i]); j++ {
			p := Point{}
			p.Ch = bc.dataNum[i][j]
			p.Fg = bc.NumColor
			p.Bg = bc.BarColor
			if bc.BarColor == ColorDefault { // the same as above
				p.Bg |= AttrReverse
			}
			if h == 0 {
				p.Bg = bc.BgColor
			}
			p.X = bc.innerX + oftX + (bc.BarWidth-len(bc.dataNum[i]))/2 + j
			p.Y = bc.innerY + bc.innerHeight - 2
			ps = append(ps, p)
		}
	}

	return bc.Block.chopOverflow(ps)
}
