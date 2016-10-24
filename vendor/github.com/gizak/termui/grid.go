// Copyright 2016 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

// GridBufferer introduces a Bufferer that can be manipulated by Grid.
type GridBufferer interface {
	Bufferer
	GetHeight() int
	SetWidth(int)
	SetX(int)
	SetY(int)
}

// Row builds a layout tree
type Row struct {
	Cols   []*Row       //children
	Widget GridBufferer // root
	X      int
	Y      int
	Width  int
	Height int
	Span   int
	Offset int
}

// calculate and set the underlying layout tree's x, y, height and width.
func (r *Row) calcLayout() {
	r.assignWidth(r.Width)
	r.Height = r.solveHeight()
	r.assignX(r.X)
	r.assignY(r.Y)
}

// tell if the node is leaf in the tree.
func (r *Row) isLeaf() bool {
	return r.Cols == nil || len(r.Cols) == 0
}

func (r *Row) isRenderableLeaf() bool {
	return r.isLeaf() && r.Widget != nil
}

// assign widgets' (and their parent rows') width recursively.
func (r *Row) assignWidth(w int) {
	r.SetWidth(w)

	accW := 0                            // acc span and offset
	calcW := make([]int, len(r.Cols))    // calculated width
	calcOftX := make([]int, len(r.Cols)) // computated start position of x

	for i, c := range r.Cols {
		accW += c.Span + c.Offset
		cw := int(float64(c.Span*r.Width) / 12.0)

		if i >= 1 {
			calcOftX[i] = calcOftX[i-1] +
				calcW[i-1] +
				int(float64(r.Cols[i-1].Offset*r.Width)/12.0)
		}

		// use up the space if it is the last col
		if i == len(r.Cols)-1 && accW == 12 {
			cw = r.Width - calcOftX[i]
		}
		calcW[i] = cw
		r.Cols[i].assignWidth(cw)
	}
}

// bottom up calc and set rows' (and their widgets') height,
// return r's total height.
func (r *Row) solveHeight() int {
	if r.isRenderableLeaf() {
		r.Height = r.Widget.GetHeight()
		return r.Widget.GetHeight()
	}

	maxh := 0
	if !r.isLeaf() {
		for _, c := range r.Cols {
			nh := c.solveHeight()
			// when embed rows in Cols, row widgets stack up
			if r.Widget != nil {
				nh += r.Widget.GetHeight()
			}
			if nh > maxh {
				maxh = nh
			}
		}
	}

	r.Height = maxh
	return maxh
}

// recursively assign x position for r tree.
func (r *Row) assignX(x int) {
	r.SetX(x)

	if !r.isLeaf() {
		acc := 0
		for i, c := range r.Cols {
			if c.Offset != 0 {
				acc += int(float64(c.Offset*r.Width) / 12.0)
			}
			r.Cols[i].assignX(x + acc)
			acc += c.Width
		}
	}
}

// recursively assign y position to r.
func (r *Row) assignY(y int) {
	r.SetY(y)

	if r.isLeaf() {
		return
	}

	for i := range r.Cols {
		acc := 0
		if r.Widget != nil {
			acc = r.Widget.GetHeight()
		}
		r.Cols[i].assignY(y + acc)
	}

}

// GetHeight implements GridBufferer interface.
func (r Row) GetHeight() int {
	return r.Height
}

// SetX implements GridBufferer interface.
func (r *Row) SetX(x int) {
	r.X = x
	if r.Widget != nil {
		r.Widget.SetX(x)
	}
}

// SetY implements GridBufferer interface.
func (r *Row) SetY(y int) {
	r.Y = y
	if r.Widget != nil {
		r.Widget.SetY(y)
	}
}

// SetWidth implements GridBufferer interface.
func (r *Row) SetWidth(w int) {
	r.Width = w
	if r.Widget != nil {
		r.Widget.SetWidth(w)
	}
}

// Buffer implements Bufferer interface,
// recursively merge all widgets buffer
func (r *Row) Buffer() Buffer {
	merged := NewBuffer()

	if r.isRenderableLeaf() {
		return r.Widget.Buffer()
	}

	// for those are not leaves but have a renderable widget
	if r.Widget != nil {
		merged.Merge(r.Widget.Buffer())
	}

	// collect buffer from children
	if !r.isLeaf() {
		for _, c := range r.Cols {
			merged.Merge(c.Buffer())
		}
	}

	return merged
}

// Grid implements 12 columns system.
// A simple example:
/*
   import ui "github.com/gizak/termui"
   // init and create widgets...

   // build
   ui.Body.AddRows(
       ui.NewRow(
           ui.NewCol(6, 0, widget0),
           ui.NewCol(6, 0, widget1)),
       ui.NewRow(
           ui.NewCol(3, 0, widget2),
           ui.NewCol(3, 0, widget30, widget31, widget32),
           ui.NewCol(6, 0, widget4)))

   // calculate layout
   ui.Body.Align()

   ui.Render(ui.Body)
*/
type Grid struct {
	Rows    []*Row
	Width   int
	X       int
	Y       int
	BgColor Attribute
}

// NewGrid returns *Grid with given rows.
func NewGrid(rows ...*Row) *Grid {
	return &Grid{Rows: rows}
}

// AddRows appends given rows to Grid.
func (g *Grid) AddRows(rs ...*Row) {
	g.Rows = append(g.Rows, rs...)
}

// NewRow creates a new row out of given columns.
func NewRow(cols ...*Row) *Row {
	rs := &Row{Span: 12, Cols: cols}
	return rs
}

// NewCol accepts: widgets are LayoutBufferer or widgets is A NewRow.
// Note that if multiple widgets are provided, they will stack up in the col.
func NewCol(span, offset int, widgets ...GridBufferer) *Row {
	r := &Row{Span: span, Offset: offset}

	if widgets != nil && len(widgets) == 1 {
		wgt := widgets[0]
		nw, isRow := wgt.(*Row)
		if isRow {
			r.Cols = nw.Cols
		} else {
			r.Widget = wgt
		}
		return r
	}

	r.Cols = []*Row{}
	ir := r
	for _, w := range widgets {
		nr := &Row{Span: 12, Widget: w}
		ir.Cols = []*Row{nr}
		ir = nr
	}

	return r
}

// Align calculate each rows' layout.
func (g *Grid) Align() {
	h := 0
	for _, r := range g.Rows {
		r.SetWidth(g.Width)
		r.SetX(g.X)
		r.SetY(g.Y + h)
		r.calcLayout()
		h += r.GetHeight()
	}
}

// Buffer implments Bufferer interface.
func (g Grid) Buffer() Buffer {
	buf := NewBuffer()

	for _, r := range g.Rows {
		buf.Merge(r.Buffer())
	}
	return buf
}

var Body *Grid
