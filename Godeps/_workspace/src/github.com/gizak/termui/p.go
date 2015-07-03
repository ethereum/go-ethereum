// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

// Par displays a paragraph.
/*
  par := termui.NewPar("Simple Text")
  par.Height = 3
  par.Width = 17
  par.Border.Label = "Label"
*/
type Par struct {
	Block
	Text        string
	TextFgColor Attribute
	TextBgColor Attribute
}

// NewPar returns a new *Par with given text as its content.
func NewPar(s string) *Par {
	return &Par{
		Block:       *NewBlock(),
		Text:        s,
		TextFgColor: theme.ParTextFg,
		TextBgColor: theme.ParTextBg}
}

// Buffer implements Bufferer interface.
func (p *Par) Buffer() []Point {
	ps := p.Block.Buffer()

	rs := str2runes(p.Text)
	i, j, k := 0, 0, 0
	for i < p.innerHeight && k < len(rs) {
		// the width of char is about to print
		w := charWidth(rs[k])

		if rs[k] == '\n' || j+w > p.innerWidth {
			i++
			j = 0 // set x = 0
			if rs[k] == '\n' {
				k++
			}

			if i >= p.innerHeight {
				ps = append(ps, newPointWithAttrs('…',
					p.innerX+p.innerWidth-1,
					p.innerY+p.innerHeight-1,
					p.TextFgColor, p.TextBgColor))
				break
			}

			continue
		}
		pi := Point{}
		pi.X = p.innerX + j
		pi.Y = p.innerY + i

		pi.Ch = rs[k]
		pi.Bg = p.TextBgColor
		pi.Fg = p.TextFgColor

		ps = append(ps, pi)

		k++
		j += w
	}
	return p.Block.chopOverflow(ps)
}
