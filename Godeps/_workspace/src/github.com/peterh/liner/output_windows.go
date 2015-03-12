package liner

import (
	"unsafe"
)

type coord struct {
	x, y int16
}
type smallRect struct {
	left, top, right, bottom int16
}

type consoleScreenBufferInfo struct {
	dwSize              coord
	dwCursorPosition    coord
	wAttributes         int16
	srWindow            smallRect
	dwMaximumWindowSize coord
}

func (s *State) cursorPos(x int) {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(s.hOut), uintptr(unsafe.Pointer(&sbi)))
	procSetConsoleCursorPosition.Call(uintptr(s.hOut),
		uintptr(int(x)&0xFFFF|int(sbi.dwCursorPosition.y)<<16))
}

func (s *State) eraseLine() {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(s.hOut), uintptr(unsafe.Pointer(&sbi)))
	var numWritten uint32
	procFillConsoleOutputCharacter.Call(uintptr(s.hOut), uintptr(' '),
		uintptr(sbi.dwSize.x-sbi.dwCursorPosition.x),
		uintptr(int(sbi.dwCursorPosition.x)&0xFFFF|int(sbi.dwCursorPosition.y)<<16),
		uintptr(unsafe.Pointer(&numWritten)))
}

func (s *State) eraseScreen() {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(s.hOut), uintptr(unsafe.Pointer(&sbi)))
	var numWritten uint32
	procFillConsoleOutputCharacter.Call(uintptr(s.hOut), uintptr(' '),
		uintptr(sbi.dwSize.x)*uintptr(sbi.dwSize.y),
		0,
		uintptr(unsafe.Pointer(&numWritten)))
	procSetConsoleCursorPosition.Call(uintptr(s.hOut), 0)
}

func (s *State) getColumns() {
	var sbi consoleScreenBufferInfo
	procGetConsoleScreenBufferInfo.Call(uintptr(s.hOut), uintptr(unsafe.Pointer(&sbi)))
	s.columns = int(sbi.dwSize.x)
}
