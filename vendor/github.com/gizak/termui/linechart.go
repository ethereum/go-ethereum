// Copyright 2016 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"fmt"
	"math"
)

// only 16 possible combinations, why bother
var braillePatterns = map[[2]int]rune{
	[2]int{0, 0}: '⣀',
	[2]int{0, 1}: '⡠',
	[2]int{0, 2}: '⡐',
	[2]int{0, 3}: '⡈',

	[2]int{1, 0}: '⢄',
	[2]int{1, 1}: '⠤',
	[2]int{1, 2}: '⠔',
	[2]int{1, 3}: '⠌',

	[2]int{2, 0}: '⢂',
	[2]int{2, 1}: '⠢',
	[2]int{2, 2}: '⠒',
	[2]int{2, 3}: '⠊',

	[2]int{3, 0}: '⢁',
	[2]int{3, 1}: '⠡',
	[2]int{3, 2}: '⠑',
	[2]int{3, 3}: '⠉',
}

var lSingleBraille = [4]rune{'\u2840', '⠄', '⠂', '⠁'}
var rSingleBraille = [4]rune{'\u2880', '⠠', '⠐', '⠈'}

// LineChart has two modes: braille(default) and dot. Using braille gives 2x capicity as dot mode,
// because one braille char can represent two data points.
/*
  lc := termui.NewLineChart()
  lc.BorderLabel = "braille-mode Line Chart"
  lc.Data = [1.2, 1.3, 1.5, 1.7, 1.5, 1.6, 1.8, 2.0]
  lc.Width = 50
  lc.Height = 12
  lc.AxesColor = termui.ColorWhite
  lc.LineColor = termui.ColorGreen | termui.AttrBold
  // termui.Render(lc)...
*/
type LineChart struct {
	Block
	Data          []float64
	DataLabels    []string // if unset, the data indices will be used
	Mode          string   // braille | dot
	DotStyle      rune
	LineColor     Attribute
	scale         float64 // data span per cell on y-axis
	AxesColor     Attribute
	drawingX      int
	drawingY      int
	axisYHeight   int
	axisXWidth    int
	axisYLabelGap int
	axisXLabelGap int
	topValue      float64
	bottomValue   float64
	labelX        [][]rune
	labelY        [][]rune
	labelYSpace   int
	maxY          float64
	minY          float64
	autoLabels    bool
}

// NewLineChart returns a new LineChart with current theme.
func NewLineChart() *LineChart {
	lc := &LineChart{Block: *NewBlock()}
	lc.AxesColor = ThemeAttr("linechart.axes.fg")
	lc.LineColor = ThemeAttr("linechart.line.fg")
	lc.Mode = "braille"
	lc.DotStyle = '•'
	lc.axisXLabelGap = 2
	lc.axisYLabelGap = 1
	lc.bottomValue = math.Inf(1)
	lc.topValue = math.Inf(-1)
	return lc
}

// one cell contains two data points
// so the capicity is 2x as dot-mode
func (lc *LineChart) renderBraille() Buffer {
	buf := NewBuffer()

	// return: b -> which cell should the point be in
	//         m -> in the cell, divided into 4 equal height levels, which subcell?
	getPos := func(d float64) (b, m int) {
		cnt4 := int((d-lc.bottomValue)/(lc.scale/4) + 0.5)
		b = cnt4 / 4
		m = cnt4 % 4
		return
	}
	// plot points
	for i := 0; 2*i+1 < len(lc.Data) && i < lc.axisXWidth; i++ {
		b0, m0 := getPos(lc.Data[2*i])
		b1, m1 := getPos(lc.Data[2*i+1])

		if b0 == b1 {
			c := Cell{
				Ch: braillePatterns[[2]int{m0, m1}],
				Bg: lc.Bg,
				Fg: lc.LineColor,
			}
			y := lc.innerArea.Min.Y + lc.innerArea.Dy() - 3 - b0
			x := lc.innerArea.Min.X + lc.labelYSpace + 1 + i
			buf.Set(x, y, c)
		} else {
			c0 := Cell{Ch: lSingleBraille[m0],
				Fg: lc.LineColor,
				Bg: lc.Bg}
			x0 := lc.innerArea.Min.X + lc.labelYSpace + 1 + i
			y0 := lc.innerArea.Min.Y + lc.innerArea.Dy() - 3 - b0
			buf.Set(x0, y0, c0)

			c1 := Cell{Ch: rSingleBraille[m1],
				Fg: lc.LineColor,
				Bg: lc.Bg}
			x1 := lc.innerArea.Min.X + lc.labelYSpace + 1 + i
			y1 := lc.innerArea.Min.Y + lc.innerArea.Dy() - 3 - b1
			buf.Set(x1, y1, c1)
		}

	}
	return buf
}

func (lc *LineChart) renderDot() Buffer {
	buf := NewBuffer()
	for i := 0; i < len(lc.Data) && i < lc.axisXWidth; i++ {
		c := Cell{
			Ch: lc.DotStyle,
			Fg: lc.LineColor,
			Bg: lc.Bg,
		}
		x := lc.innerArea.Min.X + lc.labelYSpace + 1 + i
		y := lc.innerArea.Min.Y + lc.innerArea.Dy() - 3 - int((lc.Data[i]-lc.bottomValue)/lc.scale+0.5)
		buf.Set(x, y, c)
	}

	return buf
}

func (lc *LineChart) calcLabelX() {
	lc.labelX = [][]rune{}

	for i, l := 0, 0; i < len(lc.DataLabels) && l < lc.axisXWidth; i++ {
		if lc.Mode == "dot" {
			if l >= len(lc.DataLabels) {
				break
			}

			s := str2runes(lc.DataLabels[l])
			w := strWidth(lc.DataLabels[l])
			if l+w <= lc.axisXWidth {
				lc.labelX = append(lc.labelX, s)
			}
			l += w + lc.axisXLabelGap
		} else { // braille
			if 2*l >= len(lc.DataLabels) {
				break
			}

			s := str2runes(lc.DataLabels[2*l])
			w := strWidth(lc.DataLabels[2*l])
			if l+w <= lc.axisXWidth {
				lc.labelX = append(lc.labelX, s)
			}
			l += w + lc.axisXLabelGap

		}
	}
}

func shortenFloatVal(x float64) string {
	s := fmt.Sprintf("%.2f", x)
	if len(s)-3 > 3 {
		s = fmt.Sprintf("%.2e", x)
	}

	if x < 0 {
		s = fmt.Sprintf("%.2f", x)
	}
	return s
}

func (lc *LineChart) calcLabelY() {
	span := lc.topValue - lc.bottomValue
	lc.scale = span / float64(lc.axisYHeight)

	n := (1 + lc.axisYHeight) / (lc.axisYLabelGap + 1)
	lc.labelY = make([][]rune, n)
	maxLen := 0
	for i := 0; i < n; i++ {
		s := str2runes(shortenFloatVal(lc.bottomValue + float64(i)*span/float64(n)))
		if len(s) > maxLen {
			maxLen = len(s)
		}
		lc.labelY[i] = s
	}

	lc.labelYSpace = maxLen
}

func (lc *LineChart) calcLayout() {
	// set datalabels if it is not provided
	if (lc.DataLabels == nil || len(lc.DataLabels) == 0) || lc.autoLabels {
		lc.autoLabels = true
		lc.DataLabels = make([]string, len(lc.Data))
		for i := range lc.Data {
			lc.DataLabels[i] = fmt.Sprint(i)
		}
	}

	// lazy increase, to avoid y shaking frequently
	// update bound Y when drawing is gonna overflow
	lc.minY = lc.Data[0]
	lc.maxY = lc.Data[0]

	// valid visible range
	vrange := lc.innerArea.Dx()
	if lc.Mode == "braille" {
		vrange = 2 * lc.innerArea.Dx()
	}
	if vrange > len(lc.Data) {
		vrange = len(lc.Data)
	}

	for _, v := range lc.Data[:vrange] {
		if v > lc.maxY {
			lc.maxY = v
		}
		if v < lc.minY {
			lc.minY = v
		}
	}

	span := lc.maxY - lc.minY

	if lc.minY < lc.bottomValue {
		lc.bottomValue = lc.minY - 0.2*span
	}

	if lc.maxY > lc.topValue {
		lc.topValue = lc.maxY + 0.2*span
	}

	lc.axisYHeight = lc.innerArea.Dy() - 2
	lc.calcLabelY()

	lc.axisXWidth = lc.innerArea.Dx() - 1 - lc.labelYSpace
	lc.calcLabelX()

	lc.drawingX = lc.innerArea.Min.X + 1 + lc.labelYSpace
	lc.drawingY = lc.innerArea.Min.Y
}

func (lc *LineChart) plotAxes() Buffer {
	buf := NewBuffer()

	origY := lc.innerArea.Min.Y + lc.innerArea.Dy() - 2
	origX := lc.innerArea.Min.X + lc.labelYSpace

	buf.Set(origX, origY, Cell{Ch: ORIGIN, Fg: lc.AxesColor, Bg: lc.Bg})

	for x := origX + 1; x < origX+lc.axisXWidth; x++ {
		buf.Set(x, origY, Cell{Ch: HDASH, Fg: lc.AxesColor, Bg: lc.Bg})
	}

	for dy := 1; dy <= lc.axisYHeight; dy++ {
		buf.Set(origX, origY-dy, Cell{Ch: VDASH, Fg: lc.AxesColor, Bg: lc.Bg})
	}

	// x label
	oft := 0
	for _, rs := range lc.labelX {
		if oft+len(rs) > lc.axisXWidth {
			break
		}
		for j, r := range rs {
			c := Cell{
				Ch: r,
				Fg: lc.AxesColor,
				Bg: lc.Bg,
			}
			x := origX + oft + j
			y := lc.innerArea.Min.Y + lc.innerArea.Dy() - 1
			buf.Set(x, y, c)
		}
		oft += len(rs) + lc.axisXLabelGap
	}

	// y labels
	for i, rs := range lc.labelY {
		for j, r := range rs {
			buf.Set(
				lc.innerArea.Min.X+j,
				origY-i*(lc.axisYLabelGap+1),
				Cell{Ch: r, Fg: lc.AxesColor, Bg: lc.Bg})
		}
	}

	return buf
}

// Buffer implements Bufferer interface.
func (lc *LineChart) Buffer() Buffer {
	buf := lc.Block.Buffer()

	if lc.Data == nil || len(lc.Data) == 0 {
		return buf
	}
	lc.calcLayout()
	buf.Merge(lc.plotAxes())

	if lc.Mode == "dot" {
		buf.Merge(lc.renderDot())
	} else {
		buf.Merge(lc.renderBraille())
	}

	return buf
}
