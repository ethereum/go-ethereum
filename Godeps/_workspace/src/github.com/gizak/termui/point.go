// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

// Point stands for a single cell in terminal.
type Point struct {
	Ch rune
	Bg Attribute
	Fg Attribute
	X  int
	Y  int
}

func newPoint(c rune, x, y int) (p Point) {
	p.Ch = c
	p.X = x
	p.Y = y
	return
}

func newPointWithAttrs(c rune, x, y int, fg, bg Attribute) Point {
	p := newPoint(c, x, y)
	p.Bg = bg
	p.Fg = fg
	return p
}
