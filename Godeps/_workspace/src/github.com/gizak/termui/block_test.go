package termui

import "testing"

func TestBlock_InnerBounds(t *testing.T) {
	b := NewBlock()
	b.X = 10
	b.Y = 11
	b.Width = 12
	b.Height = 13

	assert := func(name string, x, y, w, h int) {
		t.Log(name)
		cx, cy, cw, ch := b.InnerBounds()
		if cx != x {
			t.Errorf("expected x to be %d but got %d", x, cx)
		}
		if cy != y {
			t.Errorf("expected y to be %d but got %d", y, cy)
		}
		if cw != w {
			t.Errorf("expected width to be %d but got %d", w, cw)
		}
		if ch != h {
			t.Errorf("expected height to be %d but got %d", h, ch)
		}
	}

	b.HasBorder = false
	assert("no border, no padding", 10, 11, 12, 13)

	b.HasBorder = true
	assert("border, no padding", 11, 12, 10, 11)

	b.PaddingBottom = 2
	assert("border, 2b padding", 11, 12, 10, 9)

	b.PaddingTop = 3
	assert("border, 2b 3t padding", 11, 15, 10, 6)

	b.PaddingLeft = 4
	assert("border, 2b 3t 4l padding", 15, 15, 6, 6)

	b.PaddingRight = 5
	assert("border, 2b 3t 4l 5r padding", 15, 15, 1, 6)
}
