// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

/*
Package termui is a library designed for creating command line UI. For more info, goto http://github.com/gizak/termui

A simplest example:
    package main

    import ui "github.com/gizak/termui"

    func main() {
        if err:=ui.Init(); err != nil {
            panic(err)
        }
        defer ui.Close()

        g := ui.NewGauge()
        g.Percent = 50
        g.Width = 50
        g.Border.Label = "Gauge"

        ui.Render(g)
    }
*/
package termui
