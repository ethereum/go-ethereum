// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

// Block is a base struct for all other upper level widgets,
// consider it as css: display:block.
// Normally you do not need to create it manually.
type Block struct {
	X             int
	Y             int
	Border        labeledBorder
	IsDisplay     bool
	HasBorder     bool
	BgColor       Attribute
	Width         int
	Height        int
	innerWidth    int
	innerHeight   int
	innerX        int
	innerY        int
	PaddingTop    int
	PaddingBottom int
	PaddingLeft   int
	PaddingRight  int
}

// NewBlock returns a *Block which inherits styles from current theme.
func NewBlock() *Block {
	d := Block{}
	d.IsDisplay = true
	d.HasBorder = theme.HasBorder
	d.Border.BgColor = theme.BorderBg
	d.Border.FgColor = theme.BorderFg
	d.Border.LabelBgColor = theme.BorderLabelTextBg
	d.Border.LabelFgColor = theme.BorderLabelTextFg
	d.BgColor = theme.BlockBg
	d.Width = 2
	d.Height = 2
	return &d
}

// compute box model
func (d *Block) align() {
	d.innerWidth = d.Width - d.PaddingLeft - d.PaddingRight
	d.innerHeight = d.Height - d.PaddingTop - d.PaddingBottom
	d.innerX = d.X + d.PaddingLeft
	d.innerY = d.Y + d.PaddingTop

	if d.HasBorder {
		d.innerHeight -= 2
		d.innerWidth -= 2
		d.Border.X = d.X
		d.Border.Y = d.Y
		d.Border.Width = d.Width
		d.Border.Height = d.Height
		d.innerX++
		d.innerY++
	}

	if d.innerHeight < 0 {
		d.innerHeight = 0
	}
	if d.innerWidth < 0 {
		d.innerWidth = 0
	}

}

// InnerBounds returns the internal bounds of the block after aligning and
// calculating the padding and border, if any.
func (d *Block) InnerBounds() (x, y, width, height int) {
	d.align()
	return d.innerX, d.innerY, d.innerWidth, d.innerHeight
}

// Buffer implements Bufferer interface.
// Draw background and border (if any).
func (d *Block) Buffer() []Point {
	d.align()

	ps := []Point{}
	if !d.IsDisplay {
		return ps
	}

	if d.HasBorder {
		ps = d.Border.Buffer()
	}

	for i := 0; i < d.innerWidth; i++ {
		for j := 0; j < d.innerHeight; j++ {
			p := Point{}
			p.X = d.X + 1 + i
			p.Y = d.Y + 1 + j
			p.Ch = ' '
			p.Bg = d.BgColor
			ps = append(ps, p)
		}
	}
	return ps
}

// GetHeight implements GridBufferer.
// It returns current height of the block.
func (d Block) GetHeight() int {
	return d.Height
}

// SetX implements GridBufferer interface, which sets block's x position.
func (d *Block) SetX(x int) {
	d.X = x
}

// SetY implements GridBufferer interface, it sets y position for block.
func (d *Block) SetY(y int) {
	d.Y = y
}

// SetWidth implements GridBuffer interface, it sets block's width.
func (d *Block) SetWidth(w int) {
	d.Width = w
}

// chop the overflow parts
func (d *Block) chopOverflow(ps []Point) []Point {
	nps := make([]Point, 0, len(ps))
	x := d.X
	y := d.Y
	w := d.Width
	h := d.Height
	for _, v := range ps {
		if v.X >= x &&
			v.X < x+w &&
			v.Y >= y &&
			v.Y < y+h {
			nps = append(nps, v)
		}
	}
	return nps
}
