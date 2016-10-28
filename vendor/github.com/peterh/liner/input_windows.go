package liner

import (
	"bufio"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procReadConsoleInput           = kernel32.NewProc("ReadConsoleInputW")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procSetConsoleCursorPosition   = kernel32.NewProc("SetConsoleCursorPosition")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
)

// These names are from the Win32 api, so they use underscores (contrary to
// what golint suggests)
const (
	std_input_handle     = uint32(-10 & 0xFFFFFFFF)
	std_output_handle    = uint32(-11 & 0xFFFFFFFF)
	std_error_handle     = uint32(-12 & 0xFFFFFFFF)
	invalid_handle_value = ^uintptr(0)
)

type inputMode uint32

// State represents an open terminal
type State struct {
	commonState
	handle      syscall.Handle
	hOut        syscall.Handle
	origMode    inputMode
	defaultMode inputMode
	key         interface{}
	repeat      uint16
}

const (
	enableEchoInput      = 0x4
	enableInsertMode     = 0x20
	enableLineInput      = 0x2
	enableMouseInput     = 0x10
	enableProcessedInput = 0x1
	enableQuickEditMode  = 0x40
	enableWindowInput    = 0x8
)

// NewLiner initializes a new *State, and sets the terminal into raw mode. To
// restore the terminal to its previous state, call State.Close().
func NewLiner() *State {
	var s State
	hIn, _, _ := procGetStdHandle.Call(uintptr(std_input_handle))
	s.handle = syscall.Handle(hIn)
	hOut, _, _ := procGetStdHandle.Call(uintptr(std_output_handle))
	s.hOut = syscall.Handle(hOut)

	s.terminalSupported = true
	if m, err := TerminalMode(); err == nil {
		s.origMode = m.(inputMode)
		mode := s.origMode
		mode &^= enableEchoInput
		mode &^= enableInsertMode
		mode &^= enableLineInput
		mode &^= enableMouseInput
		mode |= enableWindowInput
		mode.ApplyMode()
	} else {
		s.inputRedirected = true
		s.r = bufio.NewReader(os.Stdin)
	}

	s.getColumns()
	s.outputRedirected = s.columns <= 0

	return &s
}

// These names are from the Win32 api, so they use underscores (contrary to
// what golint suggests)
const (
	focus_event              = 0x0010
	key_event                = 0x0001
	menu_event               = 0x0008
	mouse_event              = 0x0002
	window_buffer_size_event = 0x0004
)

type input_record struct {
	eventType uint16
	pad       uint16
	blob      [16]byte
}

type key_event_record struct {
	KeyDown         int32
	RepeatCount     uint16
	VirtualKeyCode  uint16
	VirtualScanCode uint16
	Char            int16
	ControlKeyState uint32
}

// These names are from the Win32 api, so they use underscores (contrary to
// what golint suggests)
const (
	vk_tab    = 0x09
	vk_prior  = 0x21
	vk_next   = 0x22
	vk_end    = 0x23
	vk_home   = 0x24
	vk_left   = 0x25
	vk_up     = 0x26
	vk_right  = 0x27
	vk_down   = 0x28
	vk_insert = 0x2d
	vk_delete = 0x2e
	vk_f1     = 0x70
	vk_f2     = 0x71
	vk_f3     = 0x72
	vk_f4     = 0x73
	vk_f5     = 0x74
	vk_f6     = 0x75
	vk_f7     = 0x76
	vk_f8     = 0x77
	vk_f9     = 0x78
	vk_f10    = 0x79
	vk_f11    = 0x7a
	vk_f12    = 0x7b
	bKey      = 0x42
	fKey      = 0x46
	yKey      = 0x59
)

const (
	shiftPressed     = 0x0010
	leftAltPressed   = 0x0002
	leftCtrlPressed  = 0x0008
	rightAltPressed  = 0x0001
	rightCtrlPressed = 0x0004

	modKeys = shiftPressed | leftAltPressed | rightAltPressed | leftCtrlPressed | rightCtrlPressed
)

func (s *State) readNext() (interface{}, error) {
	if s.repeat > 0 {
		s.repeat--
		return s.key, nil
	}

	var input input_record
	pbuf := uintptr(unsafe.Pointer(&input))
	var rv uint32
	prv := uintptr(unsafe.Pointer(&rv))

	for {
		ok, _, err := procReadConsoleInput.Call(uintptr(s.handle), pbuf, 1, prv)

		if ok == 0 {
			return nil, err
		}

		if input.eventType == window_buffer_size_event {
			xy := (*coord)(unsafe.Pointer(&input.blob[0]))
			s.columns = int(xy.x)
			if s.columns > 1 {
				s.columns--
			}
			return winch, nil
		}
		if input.eventType != key_event {
			continue
		}
		ke := (*key_event_record)(unsafe.Pointer(&input.blob[0]))
		if ke.KeyDown == 0 {
			continue
		}

		if ke.VirtualKeyCode == vk_tab && ke.ControlKeyState&modKeys == shiftPressed {
			s.key = shiftTab
		} else if ke.VirtualKeyCode == bKey && (ke.ControlKeyState&modKeys == leftAltPressed ||
			ke.ControlKeyState&modKeys == rightAltPressed) {
			s.key = altB
		} else if ke.VirtualKeyCode == fKey && (ke.ControlKeyState&modKeys == leftAltPressed ||
			ke.ControlKeyState&modKeys == rightAltPressed) {
			s.key = altF
		} else if ke.VirtualKeyCode == yKey && (ke.ControlKeyState&modKeys == leftAltPressed ||
			ke.ControlKeyState&modKeys == rightAltPressed) {
			s.key = altY
		} else if ke.Char > 0 {
			s.key = rune(ke.Char)
		} else {
			switch ke.VirtualKeyCode {
			case vk_prior:
				s.key = pageUp
			case vk_next:
				s.key = pageDown
			case vk_end:
				s.key = end
			case vk_home:
				s.key = home
			case vk_left:
				s.key = left
				if ke.ControlKeyState&(leftCtrlPressed|rightCtrlPressed) != 0 {
					if ke.ControlKeyState&modKeys == ke.ControlKeyState&(leftCtrlPressed|rightCtrlPressed) {
						s.key = wordLeft
					}
				}
			case vk_right:
				s.key = right
				if ke.ControlKeyState&(leftCtrlPressed|rightCtrlPressed) != 0 {
					if ke.ControlKeyState&modKeys == ke.ControlKeyState&(leftCtrlPressed|rightCtrlPressed) {
						s.key = wordRight
					}
				}
			case vk_up:
				s.key = up
			case vk_down:
				s.key = down
			case vk_insert:
				s.key = insert
			case vk_delete:
				s.key = del
			case vk_f1:
				s.key = f1
			case vk_f2:
				s.key = f2
			case vk_f3:
				s.key = f3
			case vk_f4:
				s.key = f4
			case vk_f5:
				s.key = f5
			case vk_f6:
				s.key = f6
			case vk_f7:
				s.key = f7
			case vk_f8:
				s.key = f8
			case vk_f9:
				s.key = f9
			case vk_f10:
				s.key = f10
			case vk_f11:
				s.key = f11
			case vk_f12:
				s.key = f12
			default:
				// Eat modifier keys
				// TODO: return Action(Unknown) if the key isn't a
				// modifier.
				continue
			}
		}

		if ke.RepeatCount > 1 {
			s.repeat = ke.RepeatCount - 1
		}
		return s.key, nil
	}
	return unknown, nil
}

// Close returns the terminal to its previous mode
func (s *State) Close() error {
	s.origMode.ApplyMode()
	return nil
}

func (s *State) startPrompt() {
	if m, err := TerminalMode(); err == nil {
		s.defaultMode = m.(inputMode)
		mode := s.defaultMode
		mode &^= enableProcessedInput
		mode.ApplyMode()
	}
}

func (s *State) restartPrompt() {
}

func (s *State) stopPrompt() {
	s.defaultMode.ApplyMode()
}

// TerminalSupported returns true because line editing is always
// supported on Windows.
func TerminalSupported() bool {
	return true
}

func (mode inputMode) ApplyMode() error {
	hIn, _, err := procGetStdHandle.Call(uintptr(std_input_handle))
	if hIn == invalid_handle_value || hIn == 0 {
		return err
	}
	ok, _, err := procSetConsoleMode.Call(hIn, uintptr(mode))
	if ok != 0 {
		err = nil
	}
	return err
}

// TerminalMode returns the current terminal input mode as an InputModeSetter.
//
// This function is provided for convenience, and should
// not be necessary for most users of liner.
func TerminalMode() (ModeApplier, error) {
	var mode inputMode
	hIn, _, err := procGetStdHandle.Call(uintptr(std_input_handle))
	if hIn == invalid_handle_value || hIn == 0 {
		return nil, err
	}
	ok, _, err := procGetConsoleMode.Call(hIn, uintptr(unsafe.Pointer(&mode)))
	if ok != 0 {
		err = nil
	}
	return mode, err
}
