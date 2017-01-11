// Copyright 2016 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"fmt"
	"sync"
)

// event mixins
type WgtMgr map[string]WgtInfo

type WgtInfo struct {
	Handlers map[string]func(Event)
	WgtRef   Widget
	Id       string
}

type Widget interface {
	Id() string
}

func NewWgtInfo(wgt Widget) WgtInfo {
	return WgtInfo{
		Handlers: make(map[string]func(Event)),
		WgtRef:   wgt,
		Id:       wgt.Id(),
	}
}

func NewWgtMgr() WgtMgr {
	wm := WgtMgr(make(map[string]WgtInfo))
	return wm

}

func (wm WgtMgr) AddWgt(wgt Widget) {
	wm[wgt.Id()] = NewWgtInfo(wgt)
}

func (wm WgtMgr) RmWgt(wgt Widget) {
	wm.RmWgtById(wgt.Id())
}

func (wm WgtMgr) RmWgtById(id string) {
	delete(wm, id)
}

func (wm WgtMgr) AddWgtHandler(id, path string, h func(Event)) {
	if w, ok := wm[id]; ok {
		w.Handlers[path] = h
	}
}

func (wm WgtMgr) RmWgtHandler(id, path string) {
	if w, ok := wm[id]; ok {
		delete(w.Handlers, path)
	}
}

var counter struct {
	sync.RWMutex
	count int
}

func GenId() string {
	counter.Lock()
	defer counter.Unlock()

	counter.count += 1
	return fmt.Sprintf("%d", counter.count)
}

func (wm WgtMgr) WgtHandlersHook() func(Event) {
	return func(e Event) {
		for _, v := range wm {
			if k := findMatch(v.Handlers, e.Path); k != "" {
				v.Handlers[k](e)
			}
		}
	}
}

var DefaultWgtMgr WgtMgr

func (b *Block) Handle(path string, handler func(Event)) {
	if _, ok := DefaultWgtMgr[b.Id()]; !ok {
		DefaultWgtMgr.AddWgt(b)
	}

	DefaultWgtMgr.AddWgtHandler(b.Id(), path, handler)
}
