// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import "image"

// Align is the position of the gauge's label.
type Align uint

// All supported positions.
const (
	AlignNone Align = 0
	AlignLeft Align = 1 << iota
	AlignRight
	AlignBottom
	AlignTop
	AlignCenterVertical
	AlignCenterHorizontal
	AlignCenter = AlignCenterVertical | AlignCenterHorizontal
)

func AlignArea(parent, child image.Rectangle, a Align) image.Rectangle {
	w, h := child.Dx(), child.Dy()

	// parent center
	pcx, pcy := parent.Min.X+parent.Dx()/2, parent.Min.Y+parent.Dy()/2
	// child center
	ccx, ccy := child.Min.X+child.Dx()/2, child.Min.Y+child.Dy()/2

	if a&AlignLeft == AlignLeft {
		child.Min.X = parent.Min.X
		child.Max.X = child.Min.X + w
	}

	if a&AlignRight == AlignRight {
		child.Max.X = parent.Max.X
		child.Min.X = child.Max.X - w
	}

	if a&AlignBottom == AlignBottom {
		child.Max.Y = parent.Max.Y
		child.Min.Y = child.Max.Y - h
	}

	if a&AlignTop == AlignRight {
		child.Min.Y = parent.Min.Y
		child.Max.Y = child.Min.Y + h
	}

	if a&AlignCenterHorizontal == AlignCenterHorizontal {
		child.Min.X += pcx - ccx
		child.Max.X = child.Min.X + w
	}

	if a&AlignCenterVertical == AlignCenterVertical {
		child.Min.Y += pcy - ccy
		child.Max.Y = child.Min.Y + h
	}

	return child
}

func MoveArea(a image.Rectangle, dx, dy int) image.Rectangle {
	a.Min.X += dx
	a.Max.X += dx
	a.Min.Y += dy
	a.Max.Y += dy
	return a
}

var termWidth int
var termHeight int

func TermRect() image.Rectangle {
	return image.Rect(0, 0, termWidth, termHeight)
}
