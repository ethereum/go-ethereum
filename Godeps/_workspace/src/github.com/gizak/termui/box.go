// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

type border struct {
	X       int
	Y       int
	Width   int
	Height  int
	FgColor Attribute
	BgColor Attribute
}

type hline struct {
	X       int
	Y       int
	Length  int
	FgColor Attribute
	BgColor Attribute
}

type vline struct {
	X       int
	Y       int
	Length  int
	FgColor Attribute
	BgColor Attribute
}

// Draw a horizontal line.
func (l hline) Buffer() []Point {
	pts := make([]Point, l.Length)
	for i := 0; i < l.Length; i++ {
		pts[i].X = l.X + i
		pts[i].Y = l.Y
		pts[i].Ch = HORIZONTAL_LINE
		pts[i].Bg = l.BgColor
		pts[i].Fg = l.FgColor
	}
	return pts
}

// Draw a vertical line.
func (l vline) Buffer() []Point {
	pts := make([]Point, l.Length)
	for i := 0; i < l.Length; i++ {
		pts[i].X = l.X
		pts[i].Y = l.Y + i
		pts[i].Ch = VERTICAL_LINE
		pts[i].Bg = l.BgColor
		pts[i].Fg = l.FgColor
	}
	return pts
}

// Draw a box border.
func (b border) Buffer() []Point {
	if b.Width < 2 || b.Height < 2 {
		return nil
	}
	pts := make([]Point, 2*b.Width+2*b.Height-4)

	pts[0].X = b.X
	pts[0].Y = b.Y
	pts[0].Fg = b.FgColor
	pts[0].Bg = b.BgColor
	pts[0].Ch = TOP_LEFT

	pts[1].X = b.X + b.Width - 1
	pts[1].Y = b.Y
	pts[1].Fg = b.FgColor
	pts[1].Bg = b.BgColor
	pts[1].Ch = TOP_RIGHT

	pts[2].X = b.X
	pts[2].Y = b.Y + b.Height - 1
	pts[2].Fg = b.FgColor
	pts[2].Bg = b.BgColor
	pts[2].Ch = BOTTOM_LEFT

	pts[3].X = b.X + b.Width - 1
	pts[3].Y = b.Y + b.Height - 1
	pts[3].Fg = b.FgColor
	pts[3].Bg = b.BgColor
	pts[3].Ch = BOTTOM_RIGHT

	copy(pts[4:], (hline{b.X + 1, b.Y, b.Width - 2, b.FgColor, b.BgColor}).Buffer())
	copy(pts[4+b.Width-2:], (hline{b.X + 1, b.Y + b.Height - 1, b.Width - 2, b.FgColor, b.BgColor}).Buffer())
	copy(pts[4+2*b.Width-4:], (vline{b.X, b.Y + 1, b.Height - 2, b.FgColor, b.BgColor}).Buffer())
	copy(pts[4+2*b.Width-4+b.Height-2:], (vline{b.X + b.Width - 1, b.Y + 1, b.Height - 2, b.FgColor, b.BgColor}).Buffer())

	return pts
}

type labeledBorder struct {
	border
	Label        string
	LabelFgColor Attribute
	LabelBgColor Attribute
}

// Draw a box border with label.
func (lb labeledBorder) Buffer() []Point {
	ps := lb.border.Buffer()
	maxTxtW := lb.Width - 2
	rs := trimStr2Runes(lb.Label, maxTxtW)

	for i, j, w := 0, 0, 0; i < len(rs); i++ {
		w = charWidth(rs[i])
		ps = append(ps, newPointWithAttrs(rs[i], lb.X+1+j, lb.Y, lb.LabelFgColor, lb.LabelBgColor))
		j += w
	}

	return ps
}
