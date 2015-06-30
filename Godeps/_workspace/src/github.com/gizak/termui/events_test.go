// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.
//
// Portions of this file uses [termbox-go](https://github.com/nsf/termbox-go/blob/54b74d087b7c397c402d0e3b66d2ccb6eaf5c2b4/api_common.go)
// by [authors](https://github.com/nsf/termbox-go/blob/master/AUTHORS)
// under [license](https://github.com/nsf/termbox-go/blob/master/LICENSE)

package termui

import (
	"errors"
	"testing"

	termbox "github.com/nsf/termbox-go"
	"github.com/stretchr/testify/assert"
)

type boxEvent termbox.Event

func TestUiEvt(t *testing.T) {
	err := errors.New("This is a mock error")
	event := boxEvent{3, 5, 2, 'H', 200, 500, err, 50, 30, 2}
	expetced := Event{3, 5, 2, 'H', 200, 500, err, 50, 30, 2}

	// We need to do that ugly casting so that vet does not complain
	assert.Equal(t, uiEvt(termbox.Event(event)), expetced)
}
