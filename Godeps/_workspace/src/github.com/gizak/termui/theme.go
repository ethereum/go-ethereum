// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

// A ColorScheme represents the current look-and-feel of the dashboard.
type ColorScheme struct {
	BodyBg            Attribute
	BlockBg           Attribute
	HasBorder         bool
	BorderFg          Attribute
	BorderBg          Attribute
	BorderLabelTextFg Attribute
	BorderLabelTextBg Attribute
	ParTextFg         Attribute
	ParTextBg         Attribute
	SparklineLine     Attribute
	SparklineTitle    Attribute
	GaugeBar          Attribute
	GaugePercent      Attribute
	LineChartLine     Attribute
	LineChartAxes     Attribute
	ListItemFg        Attribute
	ListItemBg        Attribute
	BarChartBar       Attribute
	BarChartText      Attribute
	BarChartNum       Attribute
	MBarChartBar      Attribute
	MBarChartText     Attribute
	MBarChartNum      Attribute
}

// default color scheme depends on the user's terminal setting.
var themeDefault = ColorScheme{HasBorder: true}

var themeHelloWorld = ColorScheme{
	BodyBg:            ColorBlack,
	BlockBg:           ColorBlack,
	HasBorder:         true,
	BorderFg:          ColorWhite,
	BorderBg:          ColorBlack,
	BorderLabelTextBg: ColorBlack,
	BorderLabelTextFg: ColorGreen,
	ParTextBg:         ColorBlack,
	ParTextFg:         ColorWhite,
	SparklineLine:     ColorMagenta,
	SparklineTitle:    ColorWhite,
	GaugeBar:          ColorRed,
	GaugePercent:      ColorWhite,
	LineChartLine:     ColorYellow | AttrBold,
	LineChartAxes:     ColorWhite,
	ListItemBg:        ColorBlack,
	ListItemFg:        ColorYellow,
	BarChartBar:       ColorRed,
	BarChartNum:       ColorWhite,
	BarChartText:      ColorCyan,
	MBarChartBar:      ColorRed,
	MBarChartNum:      ColorWhite,
	MBarChartText:     ColorCyan,
}

var theme = themeDefault // global dep

// Theme returns the currently used theme.
func Theme() ColorScheme {
	return theme
}

// SetTheme sets a new, custom theme.
func SetTheme(newTheme ColorScheme) {
	theme = newTheme
}

// UseTheme sets a predefined scheme. Currently available: "hello-world" and
// "black-and-white".
func UseTheme(th string) {
	switch th {
	case "helloworld":
		theme = themeHelloWorld
	default:
		theme = themeDefault
	}
}
