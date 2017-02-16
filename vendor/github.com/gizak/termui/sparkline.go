// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

// Sparkline is like: ▅▆▂▂▅▇▂▂▃▆▆▆▅▃. The data points should be non-negative integers.
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
		TitleColor: ThemeAttr("sparkline.title.fg"),
		LineColor:  ThemeAttr("sparkline.line.fg")}
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
	sl.displayWidth = sl.innerArea.Dx()

	// get how many lines gotta display
	h := 0
	sl.displayLines = 0
	for _, v := range sl.Lines {
		if h+v.displayHeight <= sl.innerArea.Dy() {
			sl.displayLines++
		} else {
			break
		}
		h += v.displayHeight
	}

	for i := 0; i < sl.displayLines; i++ {
		data := sl.Lines[i].Data

		max := 0
		for _, v := range data {
			if max < v {
				max = v
			}
		}
		sl.Lines[i].max = max
		if max != 0 {
			sl.Lines[i].scale = float32(8*sl.Lines[i].Height) / float32(max)
		} else { // when all negative
			sl.Lines[i].scale = 0
		}
	}
}

// Buffer implements Bufferer interface.
func (sl *Sparklines) Buffer() Buffer {
	buf := sl.Block.Buffer()
	sl.update()

	oftY := 0
	for i := 0; i < sl.displayLines; i++ {
		l := sl.Lines[i]
		data := l.Data

		if len(data) > sl.innerArea.Dx() {
			data = data[len(data)-sl.innerArea.Dx():]
		}

		if l.Title != "" {
			rs := trimStr2Runes(l.Title, sl.innerArea.Dx())
			oftX := 0
			for _, v := range rs {
				w := charWidth(v)
				c := Cell{
					Ch: v,
					Fg: l.TitleColor,
					Bg: sl.Bg,
				}
				x := sl.innerArea.Min.X + oftX
				y := sl.innerArea.Min.Y + oftY
				buf.Set(x, y, c)
				oftX += w
			}
		}

		for j, v := range data {
			// display height of the data point, zero when data is negative
			h := int(float32(v)*l.scale + 0.5)
			if v < 0 {
				h = 0
			}

			barCnt := h / 8
			barMod := h % 8
			for jj := 0; jj < barCnt; jj++ {
				c := Cell{
					Ch: ' ', // => sparks[7]
					Bg: l.LineColor,
				}
				x := sl.innerArea.Min.X + j
				y := sl.innerArea.Min.Y + oftY + l.Height - jj

				//p.Bg = sl.BgColor
				buf.Set(x, y, c)
			}
			if barMod != 0 {
				c := Cell{
					Ch: sparks[barMod-1],
					Fg: l.LineColor,
					Bg: sl.Bg,
				}
				x := sl.innerArea.Min.X + j
				y := sl.innerArea.Min.Y + oftY + l.Height - barCnt
				buf.Set(x, y, c)
			}
		}

		oftY += l.displayHeight
	}

	return buf
}
