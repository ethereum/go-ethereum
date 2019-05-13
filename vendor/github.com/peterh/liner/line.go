// +build windows linux darwin openbsd freebsd netbsd

package liner

import (
	"bufio"
	"container/ring"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

type action int

const (
	left action = iota
	right
	up
	down
	home
	end
	insert
	del
	pageUp
	pageDown
	f1
	f2
	f3
	f4
	f5
	f6
	f7
	f8
	f9
	f10
	f11
	f12
	altB
	altD
	altF
	altY
	shiftTab
	wordLeft
	wordRight
	winch
	unknown
)

const (
	ctrlA = 1
	ctrlB = 2
	ctrlC = 3
	ctrlD = 4
	ctrlE = 5
	ctrlF = 6
	ctrlG = 7
	ctrlH = 8
	tab   = 9
	lf    = 10
	ctrlK = 11
	ctrlL = 12
	cr    = 13
	ctrlN = 14
	ctrlO = 15
	ctrlP = 16
	ctrlQ = 17
	ctrlR = 18
	ctrlS = 19
	ctrlT = 20
	ctrlU = 21
	ctrlV = 22
	ctrlW = 23
	ctrlX = 24
	ctrlY = 25
	ctrlZ = 26
	esc   = 27
	bs    = 127
)

const (
	beep = "\a"
)

type tabDirection int

const (
	tabForward tabDirection = iota
	tabReverse
)

func (s *State) refresh(prompt []rune, buf []rune, pos int) error {
	if s.columns == 0 {
		return ErrInternal
	}

	s.needRefresh = false
	if s.multiLineMode {
		return s.refreshMultiLine(prompt, buf, pos)
	}
	return s.refreshSingleLine(prompt, buf, pos)
}

func (s *State) refreshSingleLine(prompt []rune, buf []rune, pos int) error {
	s.cursorPos(0)
	_, err := fmt.Print(string(prompt))
	if err != nil {
		return err
	}

	pLen := countGlyphs(prompt)
	bLen := countGlyphs(buf)
	// on some OS / terminals extra column is needed to place the cursor char
	if cursorColumn {
		bLen++
	}
	pos = countGlyphs(buf[:pos])
	if pLen+bLen < s.columns {
		_, err = fmt.Print(string(buf))
		s.eraseLine()
		s.cursorPos(pLen + pos)
	} else {
		// Find space available
		space := s.columns - pLen
		space-- // space for cursor
		start := pos - space/2
		end := start + space
		if end > bLen {
			end = bLen
			start = end - space
		}
		if start < 0 {
			start = 0
			end = space
		}
		pos -= start

		// Leave space for markers
		if start > 0 {
			start++
		}
		if end < bLen {
			end--
		}
		startRune := len(getPrefixGlyphs(buf, start))
		line := getPrefixGlyphs(buf[startRune:], end-start)

		// Output
		if start > 0 {
			fmt.Print("{")
		}
		fmt.Print(string(line))
		if end < bLen {
			fmt.Print("}")
		}

		// Set cursor position
		s.eraseLine()
		s.cursorPos(pLen + pos)
	}
	return err
}

func (s *State) refreshMultiLine(prompt []rune, buf []rune, pos int) error {
	promptColumns := countMultiLineGlyphs(prompt, s.columns, 0)
	totalColumns := countMultiLineGlyphs(buf, s.columns, promptColumns)
	// on some OS / terminals extra column is needed to place the cursor char
	// if cursorColumn {
	//	totalColumns++
	// }

	// it looks like Multiline mode always assume that a cursor need an extra column,
	// and always emit a newline if we are at the screen end, so no worarounds needed there

	totalRows := (totalColumns + s.columns - 1) / s.columns
	maxRows := s.maxRows
	if totalRows > s.maxRows {
		s.maxRows = totalRows
	}
	cursorRows := s.cursorRows
	if cursorRows == 0 {
		cursorRows = 1
	}

	/* First step: clear all the lines used before. To do so start by
	* going to the last row. */
	if maxRows-cursorRows > 0 {
		s.moveDown(maxRows - cursorRows)
	}

	/* Now for every row clear it, go up. */
	for i := 0; i < maxRows-1; i++ {
		s.cursorPos(0)
		s.eraseLine()
		s.moveUp(1)
	}

	/* Clean the top line. */
	s.cursorPos(0)
	s.eraseLine()

	/* Write the prompt and the current buffer content */
	if _, err := fmt.Print(string(prompt)); err != nil {
		return err
	}
	if _, err := fmt.Print(string(buf)); err != nil {
		return err
	}

	/* If we are at the very end of the screen with our prompt, we need to
	 * emit a newline and move the prompt to the first column. */
	cursorColumns := countMultiLineGlyphs(buf[:pos], s.columns, promptColumns)
	if cursorColumns == totalColumns && totalColumns%s.columns == 0 {
		s.emitNewLine()
		s.cursorPos(0)
		totalRows++
		if totalRows > s.maxRows {
			s.maxRows = totalRows
		}
	}

	/* Move cursor to right position. */
	cursorRows = (cursorColumns + s.columns) / s.columns
	if s.cursorRows > 0 && totalRows-cursorRows > 0 {
		s.moveUp(totalRows - cursorRows)
	}
	/* Set column. */
	s.cursorPos(cursorColumns % s.columns)

	s.cursorRows = cursorRows
	return nil
}

func (s *State) resetMultiLine(prompt []rune, buf []rune, pos int) {
	columns := countMultiLineGlyphs(prompt, s.columns, 0)
	columns = countMultiLineGlyphs(buf[:pos], s.columns, columns)
	columns += 2 // ^C
	cursorRows := (columns + s.columns) / s.columns
	if s.maxRows-cursorRows > 0 {
		for i := 0; i < s.maxRows-cursorRows; i++ {
			fmt.Println() // always moves the cursor down or scrolls the window up as needed
		}
	}
	s.maxRows = 1
	s.cursorRows = 0
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	longest := strs[0]

	for _, str := range strs[1:] {
		for !strings.HasPrefix(str, longest) {
			longest = longest[:len(longest)-1]
		}
	}
	// Remove trailing partial runes
	longest = strings.TrimRight(longest, "\uFFFD")
	return longest
}

func (s *State) circularTabs(items []string) func(tabDirection) (string, error) {
	item := -1
	return func(direction tabDirection) (string, error) {
		if direction == tabForward {
			if item < len(items)-1 {
				item++
			} else {
				item = 0
			}
		} else if direction == tabReverse {
			if item > 0 {
				item--
			} else {
				item = len(items) - 1
			}
		}
		return items[item], nil
	}
}

func calculateColumns(screenWidth int, items []string) (numColumns, numRows, maxWidth int) {
	for _, item := range items {
		if len(item) >= screenWidth {
			return 1, len(items), screenWidth - 1
		}
		if len(item) >= maxWidth {
			maxWidth = len(item) + 1
		}
	}

	numColumns = screenWidth / maxWidth
	numRows = len(items) / numColumns
	if len(items)%numColumns > 0 {
		numRows++
	}

	if len(items) <= numColumns {
		maxWidth = 0
	}

	return
}

func (s *State) printedTabs(items []string) func(tabDirection) (string, error) {
	numTabs := 1
	prefix := longestCommonPrefix(items)
	return func(direction tabDirection) (string, error) {
		if len(items) == 1 {
			return items[0], nil
		}

		if numTabs == 2 {
			if len(items) > 100 {
				fmt.Printf("\nDisplay all %d possibilities? (y or n) ", len(items))
			prompt:
				for {
					next, err := s.readNext()
					if err != nil {
						return prefix, err
					}

					if key, ok := next.(rune); ok {
						switch key {
						case 'n', 'N':
							return prefix, nil
						case 'y', 'Y':
							break prompt
						case ctrlC, ctrlD, cr, lf:
							s.restartPrompt()
						}
					}
				}
			}
			fmt.Println("")

			numColumns, numRows, maxWidth := calculateColumns(s.columns, items)

			for i := 0; i < numRows; i++ {
				for j := 0; j < numColumns*numRows; j += numRows {
					if i+j < len(items) {
						if maxWidth > 0 {
							fmt.Printf("%-*.[1]*s", maxWidth, items[i+j])
						} else {
							fmt.Printf("%v ", items[i+j])
						}
					}
				}
				fmt.Println("")
			}
		} else {
			numTabs++
		}
		return prefix, nil
	}
}

func (s *State) tabComplete(p []rune, line []rune, pos int) ([]rune, int, interface{}, error) {
	if s.completer == nil {
		return line, pos, rune(esc), nil
	}
	head, list, tail := s.completer(string(line), pos)
	if len(list) <= 0 {
		return line, pos, rune(esc), nil
	}
	hl := utf8.RuneCountInString(head)
	if len(list) == 1 {
		err := s.refresh(p, []rune(head+list[0]+tail), hl+utf8.RuneCountInString(list[0]))
		return []rune(head + list[0] + tail), hl + utf8.RuneCountInString(list[0]), rune(esc), err
	}

	direction := tabForward
	tabPrinter := s.circularTabs(list)
	if s.tabStyle == TabPrints {
		tabPrinter = s.printedTabs(list)
	}

	for {
		pick, err := tabPrinter(direction)
		if err != nil {
			return line, pos, rune(esc), err
		}
		err = s.refresh(p, []rune(head+pick+tail), hl+utf8.RuneCountInString(pick))
		if err != nil {
			return line, pos, rune(esc), err
		}

		next, err := s.readNext()
		if err != nil {
			return line, pos, rune(esc), err
		}
		if key, ok := next.(rune); ok {
			if key == tab {
				direction = tabForward
				continue
			}
			if key == esc {
				return line, pos, rune(esc), nil
			}
		}
		if a, ok := next.(action); ok && a == shiftTab {
			direction = tabReverse
			continue
		}
		return []rune(head + pick + tail), hl + utf8.RuneCountInString(pick), next, nil
	}
}

// reverse intelligent search, implements a bash-like history search.
func (s *State) reverseISearch(origLine []rune, origPos int) ([]rune, int, interface{}, error) {
	p := "(reverse-i-search)`': "
	err := s.refresh([]rune(p), origLine, origPos)
	if err != nil {
		return origLine, origPos, rune(esc), err
	}

	line := []rune{}
	pos := 0
	foundLine := string(origLine)
	foundPos := origPos

	getLine := func() ([]rune, []rune, int) {
		search := string(line)
		prompt := "(reverse-i-search)`%s': "
		return []rune(fmt.Sprintf(prompt, search)), []rune(foundLine), foundPos
	}

	history, positions := s.getHistoryByPattern(string(line))
	historyPos := len(history) - 1

	for {
		next, err := s.readNext()
		if err != nil {
			return []rune(foundLine), foundPos, rune(esc), err
		}

		switch v := next.(type) {
		case rune:
			switch v {
			case ctrlR: // Search backwards
				if historyPos > 0 && historyPos < len(history) {
					historyPos--
					foundLine = history[historyPos]
					foundPos = positions[historyPos]
				} else {
					fmt.Print(beep)
				}
			case ctrlS: // Search forward
				if historyPos < len(history)-1 && historyPos >= 0 {
					historyPos++
					foundLine = history[historyPos]
					foundPos = positions[historyPos]
				} else {
					fmt.Print(beep)
				}
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n

					// For each char deleted, display the last matching line of history
					history, positions := s.getHistoryByPattern(string(line))
					historyPos = len(history) - 1
					if len(history) > 0 {
						foundLine = history[historyPos]
						foundPos = positions[historyPos]
					} else {
						foundLine = ""
						foundPos = 0
					}
				}
			case ctrlG: // Cancel
				return origLine, origPos, rune(esc), err

			case tab, cr, lf, ctrlA, ctrlB, ctrlD, ctrlE, ctrlF, ctrlK,
				ctrlL, ctrlN, ctrlO, ctrlP, ctrlQ, ctrlT, ctrlU, ctrlV, ctrlW, ctrlX, ctrlY, ctrlZ:
				fallthrough
			case 0, ctrlC, esc, 28, 29, 30, 31:
				return []rune(foundLine), foundPos, next, err
			default:
				line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
				pos++

				// For each keystroke typed, display the last matching line of history
				history, positions = s.getHistoryByPattern(string(line))
				historyPos = len(history) - 1
				if len(history) > 0 {
					foundLine = history[historyPos]
					foundPos = positions[historyPos]
				} else {
					foundLine = ""
					foundPos = 0
				}
			}
		case action:
			return []rune(foundLine), foundPos, next, err
		}
		err = s.refresh(getLine())
		if err != nil {
			return []rune(foundLine), foundPos, rune(esc), err
		}
	}
}

// addToKillRing adds some text to the kill ring. If mode is 0 it adds it to a
// new node in the end of the kill ring, and move the current pointer to the new
// node. If mode is 1 or 2 it appends or prepends the text to the current entry
// of the killRing.
func (s *State) addToKillRing(text []rune, mode int) {
	// Don't use the same underlying array as text
	killLine := make([]rune, len(text))
	copy(killLine, text)

	// Point killRing to a newNode, procedure depends on the killring state and
	// append mode.
	if mode == 0 { // Add new node to killRing
		if s.killRing == nil { // if killring is empty, create a new one
			s.killRing = ring.New(1)
		} else if s.killRing.Len() >= KillRingMax { // if killring is "full"
			s.killRing = s.killRing.Next()
		} else { // Normal case
			s.killRing.Link(ring.New(1))
			s.killRing = s.killRing.Next()
		}
	} else {
		if s.killRing == nil { // if killring is empty, create a new one
			s.killRing = ring.New(1)
			s.killRing.Value = []rune{}
		}
		if mode == 1 { // Append to last entry
			killLine = append(s.killRing.Value.([]rune), killLine...)
		} else if mode == 2 { // Prepend to last entry
			killLine = append(killLine, s.killRing.Value.([]rune)...)
		}
	}

	// Save text in the current killring node
	s.killRing.Value = killLine
}

func (s *State) yank(p []rune, text []rune, pos int) ([]rune, int, interface{}, error) {
	if s.killRing == nil {
		return text, pos, rune(esc), nil
	}

	lineStart := text[:pos]
	lineEnd := text[pos:]
	var line []rune

	for {
		value := s.killRing.Value.([]rune)
		line = make([]rune, 0)
		line = append(line, lineStart...)
		line = append(line, value...)
		line = append(line, lineEnd...)

		pos = len(lineStart) + len(value)
		err := s.refresh(p, line, pos)
		if err != nil {
			return line, pos, 0, err
		}

		next, err := s.readNext()
		if err != nil {
			return line, pos, next, err
		}

		switch v := next.(type) {
		case rune:
			return line, pos, next, nil
		case action:
			switch v {
			case altY:
				s.killRing = s.killRing.Prev()
			default:
				return line, pos, next, nil
			}
		}
	}
}

// Prompt displays p and returns a line of user input, not including a trailing
// newline character. An io.EOF error is returned if the user signals end-of-file
// by pressing Ctrl-D. Prompt allows line editing if the terminal supports it.
func (s *State) Prompt(prompt string) (string, error) {
	return s.PromptWithSuggestion(prompt, "", 0)
}

// PromptWithSuggestion displays prompt and an editable text with cursor at
// given position. The cursor will be set to the end of the line if given position
// is negative or greater than length of text (in runes). Returns a line of user input, not
// including a trailing newline character. An io.EOF error is returned if the user
// signals end-of-file by pressing Ctrl-D.
func (s *State) PromptWithSuggestion(prompt string, text string, pos int) (string, error) {
	for _, r := range prompt {
		if unicode.Is(unicode.C, r) {
			return "", ErrInvalidPrompt
		}
	}
	if s.inputRedirected || !s.terminalSupported {
		return s.promptUnsupported(prompt)
	}
	p := []rune(prompt)
	const minWorkingSpace = 10
	if s.columns < countGlyphs(p)+minWorkingSpace {
		return s.tooNarrow(prompt)
	}
	if s.outputRedirected {
		return "", ErrNotTerminalOutput
	}

	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	fmt.Print(prompt)
	var line = []rune(text)
	historyEnd := ""
	var historyPrefix []string
	historyPos := 0
	historyStale := true
	historyAction := false // used to mark history related actions
	killAction := 0        // used to mark kill related actions

	defer s.stopPrompt()

	if pos < 0 || len(line) < pos {
		pos = len(line)
	}
	if len(line) > 0 {
		err := s.refresh(p, line, pos)
		if err != nil {
			return "", err
		}
	}

restart:
	s.startPrompt()
	s.getColumns()

mainLoop:
	for {
		next, err := s.readNext()
	haveNext:
		if err != nil {
			if s.shouldRestart != nil && s.shouldRestart(err) {
				goto restart
			}
			return "", err
		}

		historyAction = false
		switch v := next.(type) {
		case rune:
			switch v {
			case cr, lf:
				if s.needRefresh {
					err := s.refresh(p, line, pos)
					if err != nil {
						return "", err
					}
				}
				if s.multiLineMode {
					s.resetMultiLine(p, line, pos)
				}
				fmt.Println()
				break mainLoop
			case ctrlA: // Start of line
				pos = 0
				s.needRefresh = true
			case ctrlE: // End of line
				pos = len(line)
				s.needRefresh = true
			case ctrlB: // left
				if pos > 0 {
					pos -= len(getSuffixGlyphs(line[:pos], 1))
					s.needRefresh = true
				} else {
					fmt.Print(beep)
				}
			case ctrlF: // right
				if pos < len(line) {
					pos += len(getPrefixGlyphs(line[pos:], 1))
					s.needRefresh = true
				} else {
					fmt.Print(beep)
				}
			case ctrlD: // del
				if pos == 0 && len(line) == 0 {
					// exit
					return "", io.EOF
				}

				// ctrlD is a potential EOF, so the rune reader shuts down.
				// Therefore, if it isn't actually an EOF, we must re-startPrompt.
				s.restartPrompt()

				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					n := len(getPrefixGlyphs(line[pos:], 1))
					line = append(line[:pos], line[pos+n:]...)
					s.needRefresh = true
				}
			case ctrlK: // delete remainder of line
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					if killAction > 0 {
						s.addToKillRing(line[pos:], 1) // Add in apend mode
					} else {
						s.addToKillRing(line[pos:], 0) // Add in normal mode
					}

					killAction = 2 // Mark that there was a kill action
					line = line[:pos]
					s.needRefresh = true
				}
			case ctrlP: // up
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos > 0 {
					if historyPos == len(historyPrefix) {
						historyEnd = string(line)
					}
					historyPos--
					line = []rune(historyPrefix[historyPos])
					pos = len(line)
					s.needRefresh = true
				} else {
					fmt.Print(beep)
				}
			case ctrlN: // down
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos < len(historyPrefix) {
					historyPos++
					if historyPos == len(historyPrefix) {
						line = []rune(historyEnd)
					} else {
						line = []rune(historyPrefix[historyPos])
					}
					pos = len(line)
					s.needRefresh = true
				} else {
					fmt.Print(beep)
				}
			case ctrlT: // transpose prev glyph with glyph under cursor
				if len(line) < 2 || pos < 1 {
					fmt.Print(beep)
				} else {
					if pos == len(line) {
						pos -= len(getSuffixGlyphs(line, 1))
					}
					prev := getSuffixGlyphs(line[:pos], 1)
					next := getPrefixGlyphs(line[pos:], 1)
					scratch := make([]rune, len(prev))
					copy(scratch, prev)
					copy(line[pos-len(prev):], next)
					copy(line[pos-len(prev)+len(next):], scratch)
					pos += len(next)
					s.needRefresh = true
				}
			case ctrlL: // clear screen
				s.eraseScreen()
				s.needRefresh = true
			case ctrlC: // reset
				fmt.Println("^C")
				if s.multiLineMode {
					s.resetMultiLine(p, line, pos)
				}
				if s.ctrlCAborts {
					return "", ErrPromptAborted
				}
				line = line[:0]
				pos = 0
				fmt.Print(prompt)
				s.restartPrompt()
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n
					s.needRefresh = true
				}
			case ctrlU: // Erase line before cursor
				if killAction > 0 {
					s.addToKillRing(line[:pos], 2) // Add in prepend mode
				} else {
					s.addToKillRing(line[:pos], 0) // Add in normal mode
				}

				killAction = 2 // Mark that there was some killing
				line = line[pos:]
				pos = 0
				s.needRefresh = true
			case ctrlW: // Erase word
				if pos == 0 {
					fmt.Print(beep)
					break
				}
				// Remove whitespace to the left
				var buf []rune // Store the deleted chars in a buffer
				for {
					if pos == 0 || !unicode.IsSpace(line[pos-1]) {
						break
					}
					buf = append(buf, line[pos-1])
					line = append(line[:pos-1], line[pos:]...)
					pos--
				}
				// Remove non-whitespace to the left
				for {
					if pos == 0 || unicode.IsSpace(line[pos-1]) {
						break
					}
					buf = append(buf, line[pos-1])
					line = append(line[:pos-1], line[pos:]...)
					pos--
				}
				// Invert the buffer and save the result on the killRing
				var newBuf []rune
				for i := len(buf) - 1; i >= 0; i-- {
					newBuf = append(newBuf, buf[i])
				}
				if killAction > 0 {
					s.addToKillRing(newBuf, 2) // Add in prepend mode
				} else {
					s.addToKillRing(newBuf, 0) // Add in normal mode
				}
				killAction = 2 // Mark that there was some killing

				s.needRefresh = true
			case ctrlY: // Paste from Yank buffer
				line, pos, next, err = s.yank(p, line, pos)
				goto haveNext
			case ctrlR: // Reverse Search
				line, pos, next, err = s.reverseISearch(line, pos)
				s.needRefresh = true
				goto haveNext
			case tab: // Tab completion
				line, pos, next, err = s.tabComplete(p, line, pos)
				goto haveNext
			// Catch keys that do nothing, but you don't want them to beep
			case esc:
				// DO NOTHING
			// Unused keys
			case ctrlG, ctrlO, ctrlQ, ctrlS, ctrlV, ctrlX, ctrlZ:
				fallthrough
			// Catch unhandled control codes (anything <= 31)
			case 0, 28, 29, 30, 31:
				fmt.Print(beep)
			default:
				if pos == len(line) && !s.multiLineMode &&
					len(p)+len(line) < s.columns*4 && // Avoid countGlyphs on large lines
					countGlyphs(p)+countGlyphs(line) < s.columns-1 {
					line = append(line, v)
					fmt.Printf("%c", v)
					pos++
				} else {
					line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
					pos++
					s.needRefresh = true
				}
			}
		case action:
			switch v {
			case del:
				if pos >= len(line) {
					fmt.Print(beep)
				} else {
					n := len(getPrefixGlyphs(line[pos:], 1))
					line = append(line[:pos], line[pos+n:]...)
				}
			case left:
				if pos > 0 {
					pos -= len(getSuffixGlyphs(line[:pos], 1))
				} else {
					fmt.Print(beep)
				}
			case wordLeft, altB:
				if pos > 0 {
					var spaceHere, spaceLeft, leftKnown bool
					for {
						pos--
						if pos == 0 {
							break
						}
						if leftKnown {
							spaceHere = spaceLeft
						} else {
							spaceHere = unicode.IsSpace(line[pos])
						}
						spaceLeft, leftKnown = unicode.IsSpace(line[pos-1]), true
						if !spaceHere && spaceLeft {
							break
						}
					}
				} else {
					fmt.Print(beep)
				}
			case right:
				if pos < len(line) {
					pos += len(getPrefixGlyphs(line[pos:], 1))
				} else {
					fmt.Print(beep)
				}
			case wordRight, altF:
				if pos < len(line) {
					var spaceHere, spaceLeft, hereKnown bool
					for {
						pos++
						if pos == len(line) {
							break
						}
						if hereKnown {
							spaceLeft = spaceHere
						} else {
							spaceLeft = unicode.IsSpace(line[pos-1])
						}
						spaceHere, hereKnown = unicode.IsSpace(line[pos]), true
						if spaceHere && !spaceLeft {
							break
						}
					}
				} else {
					fmt.Print(beep)
				}
			case up:
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos > 0 {
					if historyPos == len(historyPrefix) {
						historyEnd = string(line)
					}
					historyPos--
					line = []rune(historyPrefix[historyPos])
					pos = len(line)
				} else {
					fmt.Print(beep)
				}
			case down:
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos < len(historyPrefix) {
					historyPos++
					if historyPos == len(historyPrefix) {
						line = []rune(historyEnd)
					} else {
						line = []rune(historyPrefix[historyPos])
					}
					pos = len(line)
				} else {
					fmt.Print(beep)
				}
			case home: // Start of line
				pos = 0
			case end: // End of line
				pos = len(line)
			case altD: // Delete next word
				if pos == len(line) {
					fmt.Print(beep)
					break
				}
				// Remove whitespace to the right
				var buf []rune // Store the deleted chars in a buffer
				for {
					if pos == len(line) || !unicode.IsSpace(line[pos]) {
						break
					}
					buf = append(buf, line[pos])
					line = append(line[:pos], line[pos+1:]...)
				}
				// Remove non-whitespace to the right
				for {
					if pos == len(line) || unicode.IsSpace(line[pos]) {
						break
					}
					buf = append(buf, line[pos])
					line = append(line[:pos], line[pos+1:]...)
				}
				// Save the result on the killRing
				if killAction > 0 {
					s.addToKillRing(buf, 2) // Add in prepend mode
				} else {
					s.addToKillRing(buf, 0) // Add in normal mode
				}
				killAction = 2 // Mark that there was some killing
			case winch: // Window change
				if s.multiLineMode {
					if s.maxRows-s.cursorRows > 0 {
						s.moveDown(s.maxRows - s.cursorRows)
					}
					for i := 0; i < s.maxRows-1; i++ {
						s.cursorPos(0)
						s.eraseLine()
						s.moveUp(1)
					}
					s.maxRows = 1
					s.cursorRows = 1
				}
			}
			s.needRefresh = true
		}
		if s.needRefresh && !s.inputWaiting() {
			err := s.refresh(p, line, pos)
			if err != nil {
				return "", err
			}
		}
		if !historyAction {
			historyStale = true
		}
		if killAction > 0 {
			killAction--
		}
	}
	return string(line), nil
}

// PasswordPrompt displays p, and then waits for user input. The input typed by
// the user is not displayed in the terminal.
func (s *State) PasswordPrompt(prompt string) (string, error) {
	for _, r := range prompt {
		if unicode.Is(unicode.C, r) {
			return "", ErrInvalidPrompt
		}
	}
	if !s.terminalSupported || s.columns == 0 {
		return "", errors.New("liner: function not supported in this terminal")
	}
	if s.inputRedirected {
		return s.promptUnsupported(prompt)
	}
	if s.outputRedirected {
		return "", ErrNotTerminalOutput
	}

	p := []rune(prompt)
	const minWorkingSpace = 1
	if s.columns < countGlyphs(p)+minWorkingSpace {
		return s.tooNarrow(prompt)
	}

	defer s.stopPrompt()

restart:
	s.startPrompt()
	s.getColumns()

	fmt.Print(prompt)
	var line []rune
	pos := 0

mainLoop:
	for {
		next, err := s.readNext()
		if err != nil {
			if s.shouldRestart != nil && s.shouldRestart(err) {
				goto restart
			}
			return "", err
		}

		switch v := next.(type) {
		case rune:
			switch v {
			case cr, lf:
				if s.needRefresh {
					err := s.refresh(p, line, pos)
					if err != nil {
						return "", err
					}
				}
				if s.multiLineMode {
					s.resetMultiLine(p, line, pos)
				}
				fmt.Println()
				break mainLoop
			case ctrlD: // del
				if pos == 0 && len(line) == 0 {
					// exit
					return "", io.EOF
				}

				// ctrlD is a potential EOF, so the rune reader shuts down.
				// Therefore, if it isn't actually an EOF, we must re-startPrompt.
				s.restartPrompt()
			case ctrlL: // clear screen
				s.eraseScreen()
				err := s.refresh(p, []rune{}, 0)
				if err != nil {
					return "", err
				}
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					fmt.Print(beep)
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n
				}
			case ctrlC:
				fmt.Println("^C")
				if s.multiLineMode {
					s.resetMultiLine(p, line, pos)
				}
				if s.ctrlCAborts {
					return "", ErrPromptAborted
				}
				line = line[:0]
				pos = 0
				fmt.Print(prompt)
				s.restartPrompt()
			// Unused keys
			case esc, tab, ctrlA, ctrlB, ctrlE, ctrlF, ctrlG, ctrlK, ctrlN, ctrlO, ctrlP, ctrlQ, ctrlR, ctrlS,
				ctrlT, ctrlU, ctrlV, ctrlW, ctrlX, ctrlY, ctrlZ:
				fallthrough
			// Catch unhandled control codes (anything <= 31)
			case 0, 28, 29, 30, 31:
				fmt.Print(beep)
			default:
				line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
				pos++
			}
		}
	}
	return string(line), nil
}

func (s *State) tooNarrow(prompt string) (string, error) {
	// Docker and OpenWRT and etc sometimes return 0 column width
	// Reset mode temporarily. Restore baked mode in case the terminal
	// is wide enough for the next Prompt attempt.
	m, merr := TerminalMode()
	s.origMode.ApplyMode()
	if merr == nil {
		defer m.ApplyMode()
	}
	if s.r == nil {
		// Windows does not always set s.r
		s.r = bufio.NewReader(os.Stdin)
		defer func() { s.r = nil }()
	}
	return s.promptUnsupported(prompt)
}
