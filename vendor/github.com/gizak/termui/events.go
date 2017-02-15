// Copyright 2016 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

type Event struct {
	Type string
	Path string
	From string
	To   string
	Data interface{}
	Time int64
}

var sysEvtChs []chan Event

type EvtKbd struct {
	KeyStr string
}

func evtKbd(e termbox.Event) EvtKbd {
	ek := EvtKbd{}

	k := string(e.Ch)
	pre := ""
	mod := ""

	if e.Mod == termbox.ModAlt {
		mod = "M-"
	}
	if e.Ch == 0 {
		if e.Key > 0xFFFF-12 {
			k = "<f" + strconv.Itoa(0xFFFF-int(e.Key)+1) + ">"
		} else if e.Key > 0xFFFF-25 {
			ks := []string{"<insert>", "<delete>", "<home>", "<end>", "<previous>", "<next>", "<up>", "<down>", "<left>", "<right>"}
			k = ks[0xFFFF-int(e.Key)-12]
		}

		if e.Key <= 0x7F {
			pre = "C-"
			k = string('a' - 1 + int(e.Key))
			kmap := map[termbox.Key][2]string{
				termbox.KeyCtrlSpace:     {"C-", "<space>"},
				termbox.KeyBackspace:     {"", "<backspace>"},
				termbox.KeyTab:           {"", "<tab>"},
				termbox.KeyEnter:         {"", "<enter>"},
				termbox.KeyEsc:           {"", "<escape>"},
				termbox.KeyCtrlBackslash: {"C-", "\\"},
				termbox.KeyCtrlSlash:     {"C-", "/"},
				termbox.KeySpace:         {"", "<space>"},
				termbox.KeyCtrl8:         {"C-", "8"},
			}
			if sk, ok := kmap[e.Key]; ok {
				pre = sk[0]
				k = sk[1]
			}
		}
	}

	ek.KeyStr = pre + mod + k
	return ek
}

func crtTermboxEvt(e termbox.Event) Event {
	systypemap := map[termbox.EventType]string{
		termbox.EventKey:       "keyboard",
		termbox.EventResize:    "window",
		termbox.EventMouse:     "mouse",
		termbox.EventError:     "error",
		termbox.EventInterrupt: "interrupt",
	}
	ne := Event{From: "/sys", Time: time.Now().Unix()}
	typ := e.Type
	ne.Type = systypemap[typ]

	switch typ {
	case termbox.EventKey:
		kbd := evtKbd(e)
		ne.Path = "/sys/kbd/" + kbd.KeyStr
		ne.Data = kbd
	case termbox.EventResize:
		wnd := EvtWnd{}
		wnd.Width = e.Width
		wnd.Height = e.Height
		ne.Path = "/sys/wnd/resize"
		ne.Data = wnd
	case termbox.EventError:
		err := EvtErr(e.Err)
		ne.Path = "/sys/err"
		ne.Data = err
	case termbox.EventMouse:
		m := EvtMouse{}
		m.X = e.MouseX
		m.Y = e.MouseY
		ne.Path = "/sys/mouse"
		ne.Data = m
	}
	return ne
}

type EvtWnd struct {
	Width  int
	Height int
}

type EvtMouse struct {
	X     int
	Y     int
	Press string
}

type EvtErr error

func hookTermboxEvt() {
	for {
		e := termbox.PollEvent()

		for _, c := range sysEvtChs {
			go func(ch chan Event) {
				ch <- crtTermboxEvt(e)
			}(c)
		}
	}
}

func NewSysEvtCh() chan Event {
	ec := make(chan Event)
	sysEvtChs = append(sysEvtChs, ec)
	return ec
}

var DefaultEvtStream = NewEvtStream()

type EvtStream struct {
	sync.RWMutex
	srcMap      map[string]chan Event
	stream      chan Event
	wg          sync.WaitGroup
	sigStopLoop chan Event
	Handlers    map[string]func(Event)
	hook        func(Event)
}

func NewEvtStream() *EvtStream {
	return &EvtStream{
		srcMap:      make(map[string]chan Event),
		stream:      make(chan Event),
		Handlers:    make(map[string]func(Event)),
		sigStopLoop: make(chan Event),
	}
}

func (es *EvtStream) Init() {
	es.Merge("internal", es.sigStopLoop)
	go func() {
		es.wg.Wait()
		close(es.stream)
	}()
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	return path.Clean(p)
}

func isPathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		return false
	}
	n := len(pattern)
	return len(path) >= n && path[0:n] == pattern
}

func (es *EvtStream) Merge(name string, ec chan Event) {
	es.Lock()
	defer es.Unlock()

	es.wg.Add(1)
	es.srcMap[name] = ec

	go func(a chan Event) {
		for n := range a {
			n.From = name
			es.stream <- n
		}
		es.wg.Done()
	}(ec)
}

func (es *EvtStream) Handle(path string, handler func(Event)) {
	es.Handlers[cleanPath(path)] = handler
}

func findMatch(mux map[string]func(Event), path string) string {
	n := -1
	pattern := ""
	for m := range mux {
		if !isPathMatch(m, path) {
			continue
		}
		if len(m) > n {
			pattern = m
			n = len(m)
		}
	}
	return pattern

}
// Remove all existing defined Handlers from the map
func (es *EvtStream) ResetHandlers() {
	for Path, _ := range es.Handlers {
		delete(es.Handlers, Path)
	}
	return
}

func (es *EvtStream) match(path string) string {
	return findMatch(es.Handlers, path)
}

func (es *EvtStream) Hook(f func(Event)) {
	es.hook = f
}

func (es *EvtStream) Loop() {
	for e := range es.stream {
		switch e.Path {
		case "/sig/stoploop":
			return
		}
		go func(a Event) {
			es.RLock()
			defer es.RUnlock()
			if pattern := es.match(a.Path); pattern != "" {
				es.Handlers[pattern](a)
			}
		}(e)
		if es.hook != nil {
			es.hook(e)
		}
	}
}

func (es *EvtStream) StopLoop() {
	go func() {
		e := Event{
			Path: "/sig/stoploop",
		}
		es.sigStopLoop <- e
	}()
}

func Merge(name string, ec chan Event) {
	DefaultEvtStream.Merge(name, ec)
}

func Handle(path string, handler func(Event)) {
	DefaultEvtStream.Handle(path, handler)
}

func Loop() {
	DefaultEvtStream.Loop()
}

func StopLoop() {
	DefaultEvtStream.StopLoop()
}

type EvtTimer struct {
	Duration time.Duration
	Count    uint64
}

func NewTimerCh(du time.Duration) chan Event {
	t := make(chan Event)

	go func(a chan Event) {
		n := uint64(0)
		for {
			n++
			time.Sleep(du)
			e := Event{}
			e.Type = "timer"
			e.Path = "/timer/" + du.String()
			e.Time = time.Now().Unix()
			e.Data = EvtTimer{
				Duration: du,
				Count:    n,
			}
			t <- e

		}
	}(t)
	return t
}

var DefualtHandler = func(e Event) {
}

var usrEvtCh = make(chan Event)

func SendCustomEvt(path string, data interface{}) {
	e := Event{}
	e.Path = path
	e.Data = data
	e.Time = time.Now().Unix()
	usrEvtCh <- e
}
