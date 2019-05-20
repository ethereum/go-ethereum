// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
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
  lc.Border.Label = "braille-mode Line Chart"
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
	axisYLebelGap int
	axisXLebelGap int
	topValue      float64
	bottomValue   float64
	labelX        [][]rune
	labelY        [][]rune
	labelYSpace   int
	maxY          float64
	minY          float64
}

// NewLineChart returns a new LineChart with current theme.
func NewLineChart() *LineChart {
	lc := &LineChart{Block: *NewBlock()}
	lc.AxesColor = theme.LineChartAxes
	lc.LineColor = theme.LineChartLine
	lc.Mode = "braille"
	lc.DotStyle = '•'
	lc.axisXLebelGap = 2
	lc.axisYLebelGap = 1
	lc.bottomValue = math.Inf(1)
	lc.topValue = math.Inf(-1)
	return lc
}

// one cell contains two data points
// so the capicity is 2x as dot-mode
func (lc *LineChart) renderBraille() []Point {
	ps := []Point{}

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
			p := Point{}
			p.Ch = braillePatterns[[2]int{m0, m1}]
			p.Bg = lc.BgColor
			p.Fg = lc.LineColor
			p.Y = lc.innerY + lc.innerHeight - 3 - b0
			p.X = lc.innerX + lc.labelYSpace + 1 + i
			ps = append(ps, p)
		} else {
			p0 := newPointWithAttrs(lSingleBraille[m0],
				lc.innerX+lc.labelYSpace+1+i,
				lc.innerY+lc.innerHeight-3-b0,
				lc.LineColor,
				lc.BgColor)
			p1 := newPointWithAttrs(rSingleBraille[m1],
				lc.innerX+lc.labelYSpace+1+i,
				lc.innerY+lc.innerHeight-3-b1,
				lc.LineColor,
				lc.BgColor)
			ps = append(ps, p0, p1)
		}

	}
	return ps
}

func (lc *LineChart) renderDot() []Point {
	ps := []Point{}
	for i := 0; i < len(lc.Data) && i < lc.axisXWidth; i++ {
		p := Point{}
		p.Ch = lc.DotStyle
		p.Fg = lc.LineColor
		p.Bg = lc.BgColor
		p.X = lc.innerX + lc.labelYSpace + 1 + i
		p.Y = lc.innerY + lc.innerHeight - 3 - int((lc.Data[i]-lc.bottomValue)/lc.scale+0.5)
		ps = append(ps, p)
	}

	return ps
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
			l += w + lc.axisXLebelGap
		} else { // braille
			if 2*l >= len(lc.DataLabels) {
				break
			}

			s := str2runes(lc.DataLabels[2*l])
			w := strWidth(lc.DataLabels[2*l])
			if l+w <= lc.axisXWidth {
				lc.labelX = append(lc.labelX, s)
			}
			l += w + lc.axisXLebelGap

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

	n := (1 + lc.axisYHeight) / (lc.axisYLebelGap + 1)
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
	if lc.DataLabels == nil || len(lc.DataLabels) == 0 {
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
	vrange := lc.innerWidth
	if lc.Mode == "braille" {
		vrange = 2 * lc.innerWidth
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

	lc.axisYHeight = lc.innerHeight - 2
	lc.calcLabelY()

	lc.axisXWidth = lc.innerWidth - 1 - lc.labelYSpace
	lc.calcLabelX()

	lc.drawingX = lc.innerX + 1 + lc.labelYSpace
	lc.drawingY = lc.innerY
}

func (lc *LineChart) plotAxes() []Point {
	origY := lc.innerY + lc.innerHeight - 2
	origX := lc.innerX + lc.labelYSpace

	ps := []Point{newPointWithAttrs(ORIGIN, origX, origY, lc.AxesColor, lc.BgColor)}

	for x := origX + 1; x < origX+lc.axisXWidth; x++ {
		p := Point{}
		p.X = x
		p.Y = origY
		p.Bg = lc.BgColor
		p.Fg = lc.AxesColor
		p.Ch = HDASH
		ps = append(ps, p)
	}

	for dy := 1; dy <= lc.axisYHeight; dy++ {
		p := Point{}
		p.X = origX
		p.Y = origY - dy
		p.Bg = lc.BgColor
		p.Fg = lc.AxesColor
		p.Ch = VDASH
		ps = append(ps, p)
	}

	// x label
	oft := 0
	for _, rs := range lc.labelX {
		if oft+len(rs) > lc.axisXWidth {
			break
		}
		for j, r := range rs {
			p := Point{}
			p.Ch = r
			p.Fg = lc.AxesColor
			p.Bg = lc.BgColor
			p.X = origX + oft + j
			p.Y = lc.innerY + lc.innerHeight - 1
			ps = append(ps, p)
		}
		oft += len(rs) + lc.axisXLebelGap
	}

	// y labels
	for i, rs := range lc.labelY {
		for j, r := range rs {
			p := Point{}
			p.Ch = r
			p.Fg = lc.AxesColor
			p.Bg = lc.BgColor
			p.X = lc.innerX + j
			p.Y = origY - i*(lc.axisYLebelGap+1)
			ps = append(ps, p)
		}
	}

	return ps
}

// Buffer implements Bufferer interface.
func (lc *LineChart) Buffer() []Point {
	ps := lc.Block.Buffer()
	if lc.Data == nil || len(lc.Data) == 0 {
		return ps
	}
	lc.calcLayout()
	ps = append(ps, lc.plotAxes()...)

	if lc.Mode == "dot" {
		ps = append(ps, lc.renderDot()...)
	} else {
		ps = append(ps, lc.renderBraille()...)
	}

	return lc.Block.chopOverflow(ps)
}
