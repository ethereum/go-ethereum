// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.
//
// Portions of this file uses [termbox-go](https://github.com/nsf/termbox-go/blob/54b74d087b7c397c402d0e3b66d2ccb6eaf5c2b4/api_common.go)
// by [authors](https://github.com/nsf/termbox-go/blob/master/AUTHORS)
// under [license](https://github.com/nsf/termbox-go/blob/master/LICENSE)

package termui

import "github.com/nsf/termbox-go"

/***********************************termbox-go**************************************/

type (
	EventType uint8
	Modifier  uint8
	Key       uint16
)

// This type represents a termbox event. The 'Mod', 'Key' and 'Ch' fields are
// valid if 'Type' is EventKey. The 'Width' and 'Height' fields are valid if
// 'Type' is EventResize. The 'Err' field is valid if 'Type' is EventError.
type Event struct {
	Type   EventType // one of Event* constants
	Mod    Modifier  // one of Mod* constants or 0
	Key    Key       // one of Key* constants, invalid if 'Ch' is not 0
	Ch     rune      // a unicode character
	Width  int       // width of the screen
	Height int       // height of the screen
	Err    error     // error in case if input failed
	MouseX int       // x coord of mouse
	MouseY int       // y coord of mouse
	N      int       // number of bytes written when getting a raw event
}

const (
	KeyF1 Key = 0xFFFF - iota
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyInsert
	KeyDelete
	KeyHome
	KeyEnd
	KeyPgup
	KeyPgdn
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	key_min // see terminfo
	MouseLeft
	MouseMiddle
	MouseRight
)

const (
	KeyCtrlTilde      Key = 0x00
	KeyCtrl2          Key = 0x00
	KeyCtrlSpace      Key = 0x00
	KeyCtrlA          Key = 0x01
	KeyCtrlB          Key = 0x02
	KeyCtrlC          Key = 0x03
	KeyCtrlD          Key = 0x04
	KeyCtrlE          Key = 0x05
	KeyCtrlF          Key = 0x06
	KeyCtrlG          Key = 0x07
	KeyBackspace      Key = 0x08
	KeyCtrlH          Key = 0x08
	KeyTab            Key = 0x09
	KeyCtrlI          Key = 0x09
	KeyCtrlJ          Key = 0x0A
	KeyCtrlK          Key = 0x0B
	KeyCtrlL          Key = 0x0C
	KeyEnter          Key = 0x0D
	KeyCtrlM          Key = 0x0D
	KeyCtrlN          Key = 0x0E
	KeyCtrlO          Key = 0x0F
	KeyCtrlP          Key = 0x10
	KeyCtrlQ          Key = 0x11
	KeyCtrlR          Key = 0x12
	KeyCtrlS          Key = 0x13
	KeyCtrlT          Key = 0x14
	KeyCtrlU          Key = 0x15
	KeyCtrlV          Key = 0x16
	KeyCtrlW          Key = 0x17
	KeyCtrlX          Key = 0x18
	KeyCtrlY          Key = 0x19
	KeyCtrlZ          Key = 0x1A
	KeyEsc            Key = 0x1B
	KeyCtrlLsqBracket Key = 0x1B
	KeyCtrl3          Key = 0x1B
	KeyCtrl4          Key = 0x1C
	KeyCtrlBackslash  Key = 0x1C
	KeyCtrl5          Key = 0x1D
	KeyCtrlRsqBracket Key = 0x1D
	KeyCtrl6          Key = 0x1E
	KeyCtrl7          Key = 0x1F
	KeyCtrlSlash      Key = 0x1F
	KeyCtrlUnderscore Key = 0x1F
	KeySpace          Key = 0x20
	KeyBackspace2     Key = 0x7F
	KeyCtrl8          Key = 0x7F
)

// Alt modifier constant, see Event.Mod field and SetInputMode function.
const (
	ModAlt Modifier = 0x01
)

// Event type. See Event.Type field.
const (
	EventKey EventType = iota
	EventResize
	EventMouse
	EventError
	EventInterrupt
	EventRaw
	EventNone
)

/**************************************end**************************************/

// convert termbox.Event to termui.Event
func uiEvt(e termbox.Event) Event {
	event := Event{}
	event.Type = EventType(e.Type)
	event.Mod = Modifier(e.Mod)
	event.Key = Key(e.Key)
	event.Ch = e.Ch
	event.Width = e.Width
	event.Height = e.Height
	event.Err = e.Err
	event.MouseX = e.MouseX
	event.MouseY = e.MouseY
	event.N = e.N

	return event
}

var evtChs = make([]chan Event, 0)

// EventCh returns an output-only event channel.
// This function can be called many times (multiplexer).
func EventCh() <-chan Event {
	out := make(chan Event)
	evtChs = append(evtChs, out)
	return out
}

// turn on event listener
func evtListen() {
	go func() {
		for {
			e := termbox.PollEvent()
			// dispatch
			for _, c := range evtChs {
				go func(ch chan Event) {
					ch <- uiEvt(e)
				}(c)
			}
		}
	}()
}

/*
// EventHandlers is a handler sequence
var EventHandlers []func(Event)

var signalQuit = make(chan bool)

// Quit sends quit signal to terminate termui
func Quit() {
	signalQuit <- true
}

// Wait listening to signalQuit, block operation.
func Wait() {
	<-signalQuit
}

// RegEvtHandler register function into TSEventHandler sequence.
func RegEvtHandler(fn func(Event)) {
	EventHandlers = append(EventHandlers, fn)
}

// EventLoop handles all events and
// redirects every event to callbacks in EventHandlers
func EventLoop() {
	evt := make(chan termbox.Event)

	go func() {
		for {
			evt <- termbox.PollEvent()
		}
	}()

	for {
		select {
		case c := <-signalQuit:
			defer func() { signalQuit <- c }()
			return
		case e := <-evt:
			for _, fn := range EventHandlers {
				fn(uiEvt(e))
			}
		}
	}
}
*/
