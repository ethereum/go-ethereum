/*
Package liner implements a simple command line editor, inspired by linenoise
(https://github.com/antirez/linenoise/). This package supports WIN32 in
addition to the xterm codes supported by everything else.
*/
package liner

import (
	"bufio"
	"container/ring"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"
)

type commonState struct {
	terminalSupported bool
	outputRedirected  bool
	inputRedirected   bool
	history           []string
	historyMutex      sync.RWMutex
	completer         WordCompleter
	columns           int
	killRing          *ring.Ring
	ctrlCAborts       bool
	r                 *bufio.Reader
	tabStyle          TabStyle
	multiLineMode     bool
	cursorRows        int
	maxRows           int
	shouldRestart     ShouldRestart
}

// TabStyle is used to select how tab completions are displayed.
type TabStyle int

// Two tab styles are currently available:
//
// TabCircular cycles through each completion item and displays it directly on
// the prompt
//
// TabPrints prints the list of completion items to the screen after a second
// tab key is pressed. This behaves similar to GNU readline and BASH (which
// uses readline)
const (
	TabCircular TabStyle = iota
	TabPrints
)

// ErrPromptAborted is returned from Prompt or PasswordPrompt when the user presses Ctrl-C
// if SetCtrlCAborts(true) has been called on the State
var ErrPromptAborted = errors.New("prompt aborted")

// ErrNotTerminalOutput is returned from Prompt or PasswordPrompt if the
// platform is normally supported, but stdout has been redirected
var ErrNotTerminalOutput = errors.New("standard output is not a terminal")

// Max elements to save on the killring
const KillRingMax = 60

// HistoryLimit is the maximum number of entries saved in the scrollback history.
const HistoryLimit = 1000

// ReadHistory reads scrollback history from r. Returns the number of lines
// read, and any read error (except io.EOF).
func (s *State) ReadHistory(r io.Reader) (num int, err error) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	in := bufio.NewReader(r)
	num = 0
	for {
		line, part, err := in.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return num, err
		}
		if part {
			return num, fmt.Errorf("line %d is too long", num+1)
		}
		if !utf8.Valid(line) {
			return num, fmt.Errorf("invalid string at line %d", num+1)
		}
		num++
		s.history = append(s.history, string(line))
		if len(s.history) > HistoryLimit {
			s.history = s.history[1:]
		}
	}
	return num, nil
}

// WriteHistory writes scrollback history to w. Returns the number of lines
// successfully written, and any write error.
//
// Unlike the rest of liner's API, WriteHistory is safe to call
// from another goroutine while Prompt is in progress.
// This exception is to facilitate the saving of the history buffer
// during an unexpected exit (for example, due to Ctrl-C being invoked)
func (s *State) WriteHistory(w io.Writer) (num int, err error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	for _, item := range s.history {
		_, err := fmt.Fprintln(w, item)
		if err != nil {
			return num, err
		}
		num++
	}
	return num, nil
}

// AppendHistory appends an entry to the scrollback history. AppendHistory
// should be called iff Prompt returns a valid command.
func (s *State) AppendHistory(item string) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	if len(s.history) > 0 {
		if item == s.history[len(s.history)-1] {
			return
		}
	}
	s.history = append(s.history, item)
	if len(s.history) > HistoryLimit {
		s.history = s.history[1:]
	}
}

// Returns the history lines starting with prefix
func (s *State) getHistoryByPrefix(prefix string) (ph []string) {
	for _, h := range s.history {
		if strings.HasPrefix(h, prefix) {
			ph = append(ph, h)
		}
	}
	return
}

// Returns the history lines matching the inteligent search
func (s *State) getHistoryByPattern(pattern string) (ph []string, pos []int) {
	if pattern == "" {
		return
	}
	for _, h := range s.history {
		if i := strings.Index(h, pattern); i >= 0 {
			ph = append(ph, h)
			pos = append(pos, i)
		}
	}
	return
}

// Completer takes the currently edited line content at the left of the cursor
// and returns a list of completion candidates.
// If the line is "Hello, wo!!!" and the cursor is before the first '!', "Hello, wo" is passed
// to the completer which may return {"Hello, world", "Hello, Word"} to have "Hello, world!!!".
type Completer func(line string) []string

// WordCompleter takes the currently edited line with the cursor position and
// returns the completion candidates for the partial word to be completed.
// If the line is "Hello, wo!!!" and the cursor is before the first '!', ("Hello, wo!!!", 9) is passed
// to the completer which may returns ("Hello, ", {"world", "Word"}, "!!!") to have "Hello, world!!!".
type WordCompleter func(line string, pos int) (head string, completions []string, tail string)

// SetCompleter sets the completion function that Liner will call to
// fetch completion candidates when the user presses tab.
func (s *State) SetCompleter(f Completer) {
	if f == nil {
		s.completer = nil
		return
	}
	s.completer = func(line string, pos int) (string, []string, string) {
		return "", f(string([]rune(line)[:pos])), string([]rune(line)[pos:])
	}
}

// SetWordCompleter sets the completion function that Liner will call to
// fetch completion candidates when the user presses tab.
func (s *State) SetWordCompleter(f WordCompleter) {
	s.completer = f
}

// SetTabCompletionStyle sets the behvavior when the Tab key is pressed
// for auto-completion.  TabCircular is the default behavior and cycles
// through the list of candidates at the prompt.  TabPrints will print
// the available completion candidates to the screen similar to BASH
// and GNU Readline
func (s *State) SetTabCompletionStyle(tabStyle TabStyle) {
	s.tabStyle = tabStyle
}

// ModeApplier is the interface that wraps a representation of the terminal
// mode. ApplyMode sets the terminal to this mode.
type ModeApplier interface {
	ApplyMode() error
}

// SetCtrlCAborts sets whether Prompt on a supported terminal will return an
// ErrPromptAborted when Ctrl-C is pressed. The default is false (will not
// return when Ctrl-C is pressed). Unsupported terminals typically raise SIGINT
// (and Prompt does not return) regardless of the value passed to SetCtrlCAborts.
func (s *State) SetCtrlCAborts(aborts bool) {
	s.ctrlCAborts = aborts
}

// SetMultiLineMode sets whether line is auto-wrapped. The default is false (single line).
func (s *State) SetMultiLineMode(mlmode bool) {
	s.multiLineMode = mlmode
}

// ShouldRestart is passed the error generated by readNext and returns true if
// the the read should be restarted or false if the error should be returned.
type ShouldRestart func(err error) bool

// SetShouldRestart sets the restart function that Liner will call to determine
// whether to retry the call to, or return the error returned by, readNext.
func (s *State) SetShouldRestart(f ShouldRestart) {
	s.shouldRestart = f
}

func (s *State) promptUnsupported(p string) (string, error) {
	if !s.inputRedirected || !s.terminalSupported {
		fmt.Print(p)
	}
	linebuf, _, err := s.r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(linebuf), nil
}
