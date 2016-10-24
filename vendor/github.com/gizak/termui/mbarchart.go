// Copyright 2016 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"fmt"
)

// This is the implemetation of multi-colored or stacked bar graph.  This is different from default barGraph which is implemented in bar.go
// Multi-Colored-BarChart creates multiple bars in a widget:
/*
   bc := termui.NewMBarChart()
   data := make([][]int, 2)
   data[0] := []int{3, 2, 5, 7, 9, 4}
   data[1] := []int{7, 8, 5, 3, 1, 6}
   bclabels := []string{"S0", "S1", "S2", "S3", "S4", "S5"}
   bc.BorderLabel = "Bar Chart"
   bc.Data = data
   bc.Width = 26
   bc.Height = 10
   bc.DataLabels = bclabels
   bc.TextColor = termui.ColorGreen
   bc.BarColor = termui.ColorRed
   bc.NumColor = termui.ColorYellow
*/
type MBarChart struct {
	Block
	BarColor   [NumberofColors]Attribute
	TextColor  Attribute
	NumColor   [NumberofColors]Attribute
	Data       [NumberofColors][]int
	DataLabels []string
	BarWidth   int
	BarGap     int
	labels     [][]rune
	dataNum    [NumberofColors][][]rune
	numBar     int
	scale      float64
	max        int
	minDataLen int
	numStack   int
	ShowScale  bool
	maxScale   []rune
}

// NewBarChart returns a new *BarChart with current theme.
func NewMBarChart() *MBarChart {
	bc := &MBarChart{Block: *NewBlock()}
	bc.BarColor[0] = ThemeAttr("mbarchart.bar.bg")
	bc.NumColor[0] = ThemeAttr("mbarchart.num.fg")
	bc.TextColor = ThemeAttr("mbarchart.text.fg")
	bc.BarGap = 1
	bc.BarWidth = 3
	return bc
}

func (bc *MBarChart) layout() {
	bc.numBar = bc.innerArea.Dx() / (bc.BarGap + bc.BarWidth)
	bc.labels = make([][]rune, bc.numBar)
	DataLen := 0
	LabelLen := len(bc.DataLabels)
	bc.minDataLen = 9999 //Set this to some very hight value so that we find the minimum one We want to know which array among data[][] has got the least length

	// We need to know how many stack/data array data[0] , data[1] are there
	for i := 0; i < len(bc.Data); i++ {
		if bc.Data[i] == nil {
			break
		}
		DataLen++
	}
	bc.numStack = DataLen

	//We need to know what is the mimimum size of data array data[0] could have 10 elements data[1] could have only 5, so we plot only 5 bar graphs

	for i := 0; i < DataLen; i++ {
		if bc.minDataLen > len(bc.Data[i]) {
			bc.minDataLen = len(bc.Data[i])
		}
	}

	if LabelLen > bc.minDataLen {
		LabelLen = bc.minDataLen
	}

	for i := 0; i < LabelLen && i < bc.numBar; i++ {
		bc.labels[i] = trimStr2Runes(bc.DataLabels[i], bc.BarWidth)
	}

	for i := 0; i < bc.numStack; i++ {
		bc.dataNum[i] = make([][]rune, len(bc.Data[i]))
		//For each stack of bar calcualte the rune
		for j := 0; j < LabelLen && i < bc.numBar; j++ {
			n := bc.Data[i][j]
			s := fmt.Sprint(n)
			bc.dataNum[i][j] = trimStr2Runes(s, bc.BarWidth)
		}
		//If color is not defined by default then populate a color that is different from the prevous bar
		if bc.BarColor[i] == ColorDefault && bc.NumColor[i] == ColorDefault {
			if i == 0 {
				bc.BarColor[i] = ColorBlack
			} else {
				bc.BarColor[i] = bc.BarColor[i-1] + 1
				if bc.BarColor[i] > NumberofColors {
					bc.BarColor[i] = ColorBlack
				}
			}
			bc.NumColor[i] = (NumberofColors + 1) - bc.BarColor[i] //Make NumColor opposite of barColor for visibility
		}
	}

	//If Max value is not set then we have to populate, this time the max value will be max(sum(d1[0],d2[0],d3[0]) .... sum(d1[n], d2[n], d3[n]))

	if bc.max == 0 {
		bc.max = -1
	}
	for i := 0; i < bc.minDataLen && i < LabelLen; i++ {
		var dsum int
		for j := 0; j < bc.numStack; j++ {
			dsum += bc.Data[j][i]
		}
		if dsum > bc.max {
			bc.max = dsum
		}
	}

	//Finally Calculate max sale
	if bc.ShowScale {
		s := fmt.Sprintf("%d", bc.max)
		bc.maxScale = trimStr2Runes(s, len(s))
		bc.scale = float64(bc.max) / float64(bc.innerArea.Dy()-2)
	} else {
		bc.scale = float64(bc.max) / float64(bc.innerArea.Dy()-1)
	}

}

func (bc *MBarChart) SetMax(max int) {

	if max > 0 {
		bc.max = max
	}
}

// Buffer implements Bufferer interface.
func (bc *MBarChart) Buffer() Buffer {
	buf := bc.Block.Buffer()
	bc.layout()
	var oftX int

	for i := 0; i < bc.numBar && i < bc.minDataLen && i < len(bc.DataLabels); i++ {
		ph := 0 //Previous Height to stack up
		oftX = i * (bc.BarWidth + bc.BarGap)
		for i1 := 0; i1 < bc.numStack; i1++ {
			h := int(float64(bc.Data[i1][i]) / bc.scale)
			// plot bars
			for j := 0; j < bc.BarWidth; j++ {
				for k := 0; k < h; k++ {
					c := Cell{
						Ch: ' ',
						Bg: bc.BarColor[i1],
					}
					if bc.BarColor[i1] == ColorDefault { // when color is default, space char treated as transparent!
						c.Bg |= AttrReverse
					}
					x := bc.innerArea.Min.X + i*(bc.BarWidth+bc.BarGap) + j
					y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 2 - k - ph
					buf.Set(x, y, c)

				}
			}
			ph += h
		}
		// plot text
		for j, k := 0, 0; j < len(bc.labels[i]); j++ {
			w := charWidth(bc.labels[i][j])
			c := Cell{
				Ch: bc.labels[i][j],
				Bg: bc.Bg,
				Fg: bc.TextColor,
			}
			y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 1
			x := bc.innerArea.Max.X + oftX + ((bc.BarWidth - len(bc.labels[i])) / 2) + k
			buf.Set(x, y, c)
			k += w
		}
		// plot num
		ph = 0 //re-initialize previous height
		for i1 := 0; i1 < bc.numStack; i1++ {
			h := int(float64(bc.Data[i1][i]) / bc.scale)
			for j := 0; j < len(bc.dataNum[i1][i]) && h > 0; j++ {
				c := Cell{
					Ch: bc.dataNum[i1][i][j],
					Fg: bc.NumColor[i1],
					Bg: bc.BarColor[i1],
				}
				if bc.BarColor[i1] == ColorDefault { // the same as above
					c.Bg |= AttrReverse
				}
				if h == 0 {
					c.Bg = bc.Bg
				}
				x := bc.innerArea.Min.X + oftX + (bc.BarWidth-len(bc.dataNum[i1][i]))/2 + j
				y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 2 - ph
				buf.Set(x, y, c)
			}
			ph += h
		}
	}

	if bc.ShowScale {
		//Currently bar graph only supprts data range from 0 to MAX
		//Plot 0
		c := Cell{
			Ch: '0',
			Bg: bc.Bg,
			Fg: bc.TextColor,
		}

		y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 2
		x := bc.X
		buf.Set(x, y, c)

		//Plot the maximum sacle value
		for i := 0; i < len(bc.maxScale); i++ {
			c := Cell{
				Ch: bc.maxScale[i],
				Bg: bc.Bg,
				Fg: bc.TextColor,
			}

			y := bc.innerArea.Min.Y
			x := bc.X + i

			buf.Set(x, y, c)
		}

	}

	return buf
}
