// Copyright 2016 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/gizak/termui"
	"github.com/gizak/termui/debug"
)

func main() {
	// run as client
	if len(os.Args) > 1 {
		fmt.Print(debug.ConnectAndListen())
		return
	}

	// run as server
	go func() { panic(debug.ListenAndServe()) }()

	if err := termui.Init(); err != nil {
		panic(err)
	}
	defer termui.Close()

	//termui.UseTheme("helloworld")
	b := termui.NewBlock()
	b.Width = 20
	b.Height = 20
	b.Float = termui.AlignCenter
	b.BorderLabel = "[HELLO](fg-red,bg-white) [WORLD](fg-blue,bg-green)"

	termui.Render(b)

	termui.Handle("/sys", func(e termui.Event) {
		k, ok := e.Data.(termui.EvtKbd)
		debug.Logf("->%v\n", e)
		if ok && k.KeyStr == "q" {
			termui.StopLoop()
		}
	})

	termui.Handle(("/usr"), func(e termui.Event) {
		debug.Logf("->%v\n", e)
	})

	termui.Handle("/timer/1s", func(e termui.Event) {
		t := e.Data.(termui.EvtTimer)
		termui.SendCustomEvt("/usr/t", t.Count)

		if t.Count%2 == 0 {
			b.BorderLabel = "[HELLO](fg-red,bg-green) [WORLD](fg-blue,bg-white)"
		} else {
			b.BorderLabel = "[HELLO](fg-blue,bg-white) [WORLD](fg-red,bg-green)"
		}

		termui.Render(b)

	})

	termui.Loop()
}
