// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import tm "github.com/nsf/termbox-go"

// Bufferer should be implemented by all renderable components.
type Bufferer interface {
	Buffer() []Point
}

// Init initializes termui library. This function should be called before any others.
// After initialization, the library must be finalized by 'Close' function.
func Init() error {
	Body = NewGrid()
	Body.X = 0
	Body.Y = 0
	Body.BgColor = theme.BodyBg
	defer func() {
		w, _ := tm.Size()
		Body.Width = w
		evtListen()
	}()
	return tm.Init()
}

// Close finalizes termui library,
// should be called after successful initialization when termui's functionality isn't required anymore.
func Close() {
	tm.Close()
}

// TermWidth returns the current terminal's width.
func TermWidth() int {
	tm.Sync()
	w, _ := tm.Size()
	return w
}

// TermHeight returns the current terminal's height.
func TermHeight() int {
	tm.Sync()
	_, h := tm.Size()
	return h
}

// Render renders all Bufferer in the given order from left to right,
// right could overlap on left ones.
func Render(rs ...Bufferer) {
	tm.Clear(tm.ColorDefault, toTmAttr(theme.BodyBg))
	for _, r := range rs {
		buf := r.Buffer()
		for _, v := range buf {
			tm.SetCell(v.X, v.Y, v.Ch, toTmAttr(v.Fg), toTmAttr(v.Bg))
		}
	}
	tm.Flush()
}
