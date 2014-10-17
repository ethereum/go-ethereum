/* Inspired by https://github.com/xuyu/logging/blob/master/colorful_win.go */

package ethrepl

import (
	"syscall"
	"unsafe"
)

type color uint16

const (
	green  = color(0x0002)
	red    = color(0x0004)
	yellow = color(0x000E)
)

const (
	mask = uint16(yellow | green | red)
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procSetConsoleTextAttribute    = kernel32.NewProc("SetConsoleTextAttribute")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	hStdout                        uintptr
	initScreenInfo                 *consoleScreenBufferInfo
)

func setConsoleTextAttribute(hConsoleOutput uintptr, wAttributes uint16) bool {
	ret, _, _ := procSetConsoleTextAttribute.Call(hConsoleOutput, uintptr(wAttributes))
	return ret != 0
}

type coord struct {
	X, Y int16
}

type smallRect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	DwSize              coord
	DwCursorPosition    coord
	WAttributes         uint16
	SrWindow            smallRect
	DwMaximumWindowSize coord
}

func getConsoleScreenBufferInfo(hConsoleOutput uintptr) *consoleScreenBufferInfo {
	var csbi consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(hConsoleOutput, uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 {
		return nil
	}
	return &csbi
}

const (
	stdOutputHandle = uint32(-11 & 0xFFFFFFFF)
)

func init() {
	hStdout, _, _ = procGetStdHandle.Call(uintptr(stdOutputHandle))
	initScreenInfo = getConsoleScreenBufferInfo(hStdout)
}

func resetColorful() {
	if initScreenInfo == nil {
		return
	}
	setConsoleTextAttribute(hStdout, initScreenInfo.WAttributes)
}

func changeColor(c color) {
	attr := uint16(0) & ^mask | uint16(c)
	setConsoleTextAttribute(hStdout, attr)
}
