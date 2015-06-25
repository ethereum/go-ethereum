// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"strconv"
	"strings"
)

// Gauge is a progress bar like widget.
// A simple example:
/*
  g := termui.NewGauge()
  g.Percent = 40
  g.Width = 50
  g.Height = 3
  g.Border.Label = "Slim Gauge"
  g.BarColor = termui.ColorRed
  g.PercentColor = termui.ColorBlue
*/

// Align is the position of the gauge's label.
type Align int

// All supported positions.
const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

type Gauge struct {
	Block
	Percent      int
	BarColor     Attribute
	PercentColor Attribute
	Label        string
	LabelAlign   Align
}

// NewGauge return a new gauge with current theme.
func NewGauge() *Gauge {
	g := &Gauge{
		Block:        *NewBlock(),
		PercentColor: theme.GaugePercent,
		BarColor:     theme.GaugeBar,
		Label:        "{{percent}}%",
		LabelAlign:   AlignCenter,
	}

	g.Width = 12
	g.Height = 5
	return g
}

// Buffer implements Bufferer interface.
func (g *Gauge) Buffer() []Point {
	ps := g.Block.Buffer()

	// plot bar
	w := g.Percent * g.innerWidth / 100
	for i := 0; i < g.innerHeight; i++ {
		for j := 0; j < w; j++ {
			p := Point{}
			p.X = g.innerX + j
			p.Y = g.innerY + i
			p.Ch = ' '
			p.Bg = g.BarColor
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}
			ps = append(ps, p)
		}
	}

	// plot percentage
	s := strings.Replace(g.Label, "{{percent}}", strconv.Itoa(g.Percent), -1)
	pry := g.innerY + g.innerHeight/2
	rs := str2runes(s)
	var pos int
	switch g.LabelAlign {
	case AlignLeft:
		pos = 0

	case AlignCenter:
		pos = (g.innerWidth - strWidth(s)) / 2

	case AlignRight:
		pos = g.innerWidth - strWidth(s)
	}

	for i, v := range rs {
		p := Point{}
		p.X = 1 + pos + i
		p.Y = pry
		p.Ch = v
		p.Fg = g.PercentColor
		if w+g.innerX > pos+i {
			p.Bg = g.BarColor
			if p.Bg == ColorDefault {
				p.Bg |= AttrReverse
			}

		} else {
			p.Bg = g.Block.BgColor
		}

		ps = append(ps, p)
	}
	return g.Block.chopOverflow(ps)
}
