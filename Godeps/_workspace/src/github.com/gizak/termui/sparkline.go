// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import "math"

// Sparkline is like: ▅▆▂▂▅▇▂▂▃▆▆▆▅▃
/*
  data := []int{4, 2, 1, 6, 3, 9, 1, 4, 2, 15, 14, 9, 8, 6, 10, 13, 15, 12, 10, 5, 3, 6, 1}
  spl := termui.NewSparkline()
  spl.Data = data
  spl.Title = "Sparkline 0"
  spl.LineColor = termui.ColorGreen
*/
type Sparkline struct {
	Data          []int
	Height        int
	Title         string
	TitleColor    Attribute
	LineColor     Attribute
	displayHeight int
	scale         float32
	max           int
}

// Sparklines is a renderable widget which groups together the given sparklines.
/*
  spls := termui.NewSparklines(spl0,spl1,spl2) //...
  spls.Height = 2
  spls.Width = 20
*/
type Sparklines struct {
	Block
	Lines        []Sparkline
	displayLines int
	displayWidth int
}

var sparks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Add appends a given Sparkline to s *Sparklines.
func (s *Sparklines) Add(sl Sparkline) {
	s.Lines = append(s.Lines, sl)
}

// NewSparkline returns a unrenderable single sparkline that intended to be added into Sparklines.
func NewSparkline() Sparkline {
	return Sparkline{
		Height:     1,
		TitleColor: theme.SparklineTitle,
		LineColor:  theme.SparklineLine}
}

// NewSparklines return a new *Spaklines with given Sparkline(s), you can always add a new Sparkline later.
func NewSparklines(ss ...Sparkline) *Sparklines {
	s := &Sparklines{Block: *NewBlock(), Lines: ss}
	return s
}

func (sl *Sparklines) update() {
	for i, v := range sl.Lines {
		if v.Title == "" {
			sl.Lines[i].displayHeight = v.Height
		} else {
			sl.Lines[i].displayHeight = v.Height + 1
		}
	}
	sl.displayWidth = sl.innerWidth

	// get how many lines gotta display
	h := 0
	sl.displayLines = 0
	for _, v := range sl.Lines {
		if h+v.displayHeight <= sl.innerHeight {
			sl.displayLines++
		} else {
			break
		}
		h += v.displayHeight
	}

	for i := 0; i < sl.displayLines; i++ {
		data := sl.Lines[i].Data

		max := math.MinInt32
		for _, v := range data {
			if max < v {
				max = v
			}
		}
		sl.Lines[i].max = max
		sl.Lines[i].scale = float32(8*sl.Lines[i].Height) / float32(max)
	}
}

// Buffer implements Bufferer interface.
func (sl *Sparklines) Buffer() []Point {
	ps := sl.Block.Buffer()
	sl.update()

	oftY := 0
	for i := 0; i < sl.displayLines; i++ {
		l := sl.Lines[i]
		data := l.Data

		if len(data) > sl.innerWidth {
			data = data[len(data)-sl.innerWidth:]
		}

		if l.Title != "" {
			rs := trimStr2Runes(l.Title, sl.innerWidth)
			oftX := 0
			for _, v := range rs {
				w := charWidth(v)
				p := Point{}
				p.Ch = v
				p.Fg = l.TitleColor
				p.Bg = sl.BgColor
				p.X = sl.innerX + oftX
				p.Y = sl.innerY + oftY
				ps = append(ps, p)
				oftX += w
			}
		}

		for j, v := range data {
			h := int(float32(v)*l.scale + 0.5)
			barCnt := h / 8
			barMod := h % 8
			for jj := 0; jj < barCnt; jj++ {
				p := Point{}
				p.X = sl.innerX + j
				p.Y = sl.innerY + oftY + l.Height - jj
				p.Ch = ' ' // => sparks[7]
				p.Bg = l.LineColor
				//p.Bg = sl.BgColor
				ps = append(ps, p)
			}
			if barMod != 0 {
				p := Point{}
				p.X = sl.innerX + j
				p.Y = sl.innerY + oftY + l.Height - barCnt
				p.Ch = sparks[barMod-1]
				p.Fg = l.LineColor
				p.Bg = sl.BgColor
				ps = append(ps, p)
			}
		}

		oftY += l.displayHeight
	}

	return sl.Block.chopOverflow(ps)
}
