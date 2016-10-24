# termui [![Build Status](https://travis-ci.org/gizak/termui.svg?branch=master)](https://travis-ci.org/gizak/termui) [![Doc Status](https://godoc.org/github.com/gizak/termui?status.png)](https://godoc.org/github.com/gizak/termui)

<img src="./_example/dashboard.gif" alt="demo cast under osx 10.10; Terminal.app; Menlo Regular 12pt.)" width="80%">

`termui` is a cross-platform, easy-to-compile, and fully-customizable terminal dashboard. It is inspired by [blessed-contrib](https://github.com/yaronn/blessed-contrib), but purely in Go.

Now version v2 has arrived! It brings new event system, new theme system, new `Buffer` interface and specific colour text rendering. (some docs are missing, but it will be completed soon!)

## Installation

`master` mirrors v2 branch, to install:

	go get -u github.com/gizak/termui

It is recommanded to use locked deps by using [glide](https://glide.sh): move to `termui` src directory then run `glide up`.

For the compatible reason, you can choose to install the legacy version of `termui`:

	go get gopkg.in/gizak/termui.v1

## Usage

### Layout

To use `termui`, the very first thing you may want to know is how to manage layout. `termui` offers two ways of doing this, known as absolute layout and grid layout.

__Absolute layout__

Each widget has an underlying block structure which basically is a box model. It has border, label and padding properties. A border of a widget can be chosen to hide or display (with its border label), you can pick a different front/back colour for the border as well. To display such a widget at a specific location in terminal window, you need to assign `.X`, `.Y`, `.Height`, `.Width` values for each widget before sending it to `.Render`. Let's demonstrate these by a code snippet:

`````go
	import ui "github.com/gizak/termui" // <- ui shortcut, optional

	func main() {
		err := ui.Init()
		if err != nil {
			panic(err)
		}
		defer ui.Close()

		p := ui.NewPar(":PRESS q TO QUIT DEMO")
		p.Height = 3
		p.Width = 50
		p.TextFgColor = ui.ColorWhite
		p.BorderLabel = "Text Box"
		p.BorderFg = ui.ColorCyan

		g := ui.NewGauge()
		g.Percent = 50
		g.Width = 50
		g.Height = 3
		g.Y = 11
		g.BorderLabel = "Gauge"
		g.BarColor = ui.ColorRed
		g.BorderFg = ui.ColorWhite
		g.BorderLabelFg = ui.ColorCyan

		ui.Render(p, g) // feel free to call Render, it's async and non-block

		// event handler...
	}
`````

Note that components can be overlapped (I'd rather call this a feature...), `Render(rs ...Renderer)` renders its args from left to right (i.e. each component's weight is arising from left to right).

__Grid layout:__

<img src="./_example/grid.gif" alt="grid" width="60%">

Grid layout uses [12 columns grid system](http://www.w3schools.com/bootstrap/bootstrap_grid_system.asp) with expressive syntax. To use `Grid`, all we need to do is build a widget tree consisting of `Row`s and `Col`s (Actually a `Col` is also a `Row` but with a widget endpoint attached).

```go
	import ui "github.com/gizak/termui"
	// init and create widgets...

	// build
	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(6, 0, widget0),
			ui.NewCol(6, 0, widget1)),
		ui.NewRow(
			ui.NewCol(3, 0, widget2),
			ui.NewCol(3, 0, widget30, widget31, widget32),
			ui.NewCol(6, 0, widget4)))

	// calculate layout
	ui.Body.Align()

	ui.Render(ui.Body)
```

### Events

`termui` ships with a http-like event mux handling system. All events are channeled up from different sources (typing, click, windows resize, custom event) and then encoded as universal `Event` object. `Event.Path` indicates the event type and `Event.Data` stores the event data struct. Add a handler to a certain event is easy as below:

```go
	// handle key q pressing
	ui.Handle("/sys/kbd/q", func(ui.Event) {
		// press q to quit
		ui.StopLoop()
	})

	ui.Handle("/sys/kbd/C-x", func(ui.Event) {
		// handle Ctrl + x combination
	})

	ui.Handle("/sys/kbd", func(ui.Event) {
		// handle all other key pressing
	})

	// handle a 1s timer
	ui.Handle("/timer/1s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		// t is a EvtTimer
		if t.Count%2 ==0 {
			// do something
		}
	})

	ui.Loop() // block until StopLoop is called
```

### Widgets

Click image to see the corresponding demo codes.

[<img src="./_example/par.png" alt="par" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/par.go)
[<img src="./_example/list.png" alt="list" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/list.go)
[<img src="./_example/gauge.png" alt="gauge" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/gauge.go)
[<img src="./_example/linechart.png" alt="linechart" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/linechart.go)
[<img src="./_example/barchart.png" alt="barchart" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/barchart.go)
[<img src="./_example/mbarchart.png" alt="barchart" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/mbarchart.go)
[<img src="./_example/sparklines.png" alt="sparklines" type="image/png" width="45%">](https://github.com/gizak/termui/blob/master/_example/sparklines.go)

## GoDoc

[godoc](https://godoc.org/github.com/gizak/termui)

## TODO

- [x] Grid layout
- [x] Event system
- [x] Canvas widget
- [x] Refine APIs
- [ ] Focusable widgets

## Changelog

## License
This library is under the [MIT License](http://opensource.org/licenses/MIT)
