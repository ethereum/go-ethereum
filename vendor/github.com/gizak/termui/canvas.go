// Copyright 2016 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

/*
dots:
   ,___,
   |1 4|
   |2 5|
   |3 6|
   |7 8|
   `````
*/

var brailleBase = '\u2800'

var brailleOftMap = [4][2]rune{
	{'\u0001', '\u0008'},
	{'\u0002', '\u0010'},
	{'\u0004', '\u0020'},
	{'\u0040', '\u0080'}}

// Canvas contains drawing map: i,j -> rune
type Canvas map[[2]int]rune

// NewCanvas returns an empty Canvas
func NewCanvas() Canvas {
	return make(map[[2]int]rune)
}

func chOft(x, y int) rune {
	return brailleOftMap[y%4][x%2]
}

func (c Canvas) rawCh(x, y int) rune {
	if ch, ok := c[[2]int{x, y}]; ok {
		return ch
	}
	return '\u0000' //brailleOffset
}

// return coordinate in terminal
func chPos(x, y int) (int, int) {
	return y / 4, x / 2
}

// Set sets a point (x,y) in the virtual coordinate
func (c Canvas) Set(x, y int) {
	i, j := chPos(x, y)
	ch := c.rawCh(i, j)
	ch |= chOft(x, y)
	c[[2]int{i, j}] = ch
}

// Unset removes point (x,y)
func (c Canvas) Unset(x, y int) {
	i, j := chPos(x, y)
	ch := c.rawCh(i, j)
	ch &= ^chOft(x, y)
	c[[2]int{i, j}] = ch
}

// Buffer returns un-styled points
func (c Canvas) Buffer() Buffer {
	buf := NewBuffer()
	for k, v := range c {
		buf.Set(k[0], k[1], Cell{Ch: v + brailleBase})
	}
	return buf
}
