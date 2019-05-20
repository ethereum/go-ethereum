// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import "strings"

// List displays []string as its items,
// it has a Overflow option (default is "hidden"), when set to "hidden",
// the item exceeding List's width is truncated, but when set to "wrap",
// the overflowed text breaks into next line.
/*
  strs := []string{
		"[0] github.com/gizak/termui",
		"[1] editbox.go",
		"[2] iterrupt.go",
		"[3] keyboard.go",
		"[4] output.go",
		"[5] random_out.go",
		"[6] dashboard.go",
		"[7] nsf/termbox-go"}

  ls := termui.NewList()
  ls.Items = strs
  ls.ItemFgColor = termui.ColorYellow
  ls.Border.Label = "List"
  ls.Height = 7
  ls.Width = 25
  ls.Y = 0
*/
type List struct {
	Block
	Items       []string
	Overflow    string
	ItemFgColor Attribute
	ItemBgColor Attribute
}

// NewList returns a new *List with current theme.
func NewList() *List {
	l := &List{Block: *NewBlock()}
	l.Overflow = "hidden"
	l.ItemFgColor = theme.ListItemFg
	l.ItemBgColor = theme.ListItemBg
	return l
}

// Buffer implements Bufferer interface.
func (l *List) Buffer() []Point {
	ps := l.Block.Buffer()
	switch l.Overflow {
	case "wrap":
		rs := str2runes(strings.Join(l.Items, "\n"))
		i, j, k := 0, 0, 0
		for i < l.innerHeight && k < len(rs) {
			w := charWidth(rs[k])
			if rs[k] == '\n' || j+w > l.innerWidth {
				i++
				j = 0
				if rs[k] == '\n' {
					k++
				}
				continue
			}
			pi := Point{}
			pi.X = l.innerX + j
			pi.Y = l.innerY + i

			pi.Ch = rs[k]
			pi.Bg = l.ItemBgColor
			pi.Fg = l.ItemFgColor

			ps = append(ps, pi)
			k++
			j++
		}

	case "hidden":
		trimItems := l.Items
		if len(trimItems) > l.innerHeight {
			trimItems = trimItems[:l.innerHeight]
		}
		for i, v := range trimItems {
			rs := trimStr2Runes(v, l.innerWidth)

			j := 0
			for _, vv := range rs {
				w := charWidth(vv)
				p := Point{}
				p.X = l.innerX + j
				p.Y = l.innerY + i

				p.Ch = vv
				p.Bg = l.ItemBgColor
				p.Fg = l.ItemFgColor

				ps = append(ps, p)
				j += w
			}
		}
	}
	return l.Block.chopOverflow(ps)
}
