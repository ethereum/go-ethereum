package termui

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestCanvasSet(t *testing.T) {
	c := NewCanvas()
	c.Set(0, 0)
	c.Set(0, 1)
	c.Set(0, 2)
	c.Set(0, 3)
	c.Set(1, 3)
	c.Set(2, 3)
	c.Set(3, 3)
	c.Set(4, 3)
	c.Set(5, 3)
	spew.Dump(c)
}

func TestCanvasUnset(t *testing.T) {
	c := NewCanvas()
	c.Set(0, 0)
	c.Set(0, 1)
	c.Set(0, 2)
	c.Unset(0, 2)
	spew.Dump(c)
	c.Unset(0, 3)
	spew.Dump(c)
}

func TestCanvasBuffer(t *testing.T) {
	c := NewCanvas()
	c.Set(0, 0)
	c.Set(0, 1)
	c.Set(0, 2)
	c.Set(0, 3)
	c.Set(1, 3)
	c.Set(2, 3)
	c.Set(3, 3)
	c.Set(4, 3)
	c.Set(5, 3)
	c.Set(6, 3)
	c.Set(7, 2)
	c.Set(8, 1)
	c.Set(9, 0)
	bufs := c.Buffer()
	rs := make([]rune, len(bufs))
	for i, v := range bufs {
		rs[i] = v.Ch
	}
	spew.Dump(string(rs))
}
