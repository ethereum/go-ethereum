// Copyright 2016 Zack Guo <gizak@icloud.com>. All rights reserved.
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
	l.ItemFgColor = ThemeAttr("list.item.fg")
	l.ItemBgColor = ThemeAttr("list.item.bg")
	return l
}

// Buffer implements Bufferer interface.
func (l *List) Buffer() Buffer {
	buf := l.Block.Buffer()

	switch l.Overflow {
	case "wrap":
		cs := DefaultTxBuilder.Build(strings.Join(l.Items, "\n"), l.ItemFgColor, l.ItemBgColor)
		i, j, k := 0, 0, 0
		for i < l.innerArea.Dy() && k < len(cs) {
			w := cs[k].Width()
			if cs[k].Ch == '\n' || j+w > l.innerArea.Dx() {
				i++
				j = 0
				if cs[k].Ch == '\n' {
					k++
				}
				continue
			}
			buf.Set(l.innerArea.Min.X+j, l.innerArea.Min.Y+i, cs[k])

			k++
			j++
		}

	case "hidden":
		trimItems := l.Items
		if len(trimItems) > l.innerArea.Dy() {
			trimItems = trimItems[:l.innerArea.Dy()]
		}
		for i, v := range trimItems {
			cs := DTrimTxCls(DefaultTxBuilder.Build(v, l.ItemFgColor, l.ItemBgColor), l.innerArea.Dx())
			j := 0
			for _, vv := range cs {
				w := vv.Width()
				buf.Set(l.innerArea.Min.X+j, l.innerArea.Min.Y+i, vv)
				j += w
			}
		}
	}
	return buf
}
