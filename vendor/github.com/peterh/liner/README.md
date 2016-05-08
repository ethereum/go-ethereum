Liner
=====

Liner is a command line editor with history. It was inspired by linenoise;
everything Unix-like is a VT100 (or is trying very hard to be). If your
terminal is not pretending to be a VT100, change it. Liner also support
Windows.

Liner is released under the X11 license (which is similar to the new BSD
license).

Line Editing
------------

The following line editing commands are supported on platforms and terminals
that Liner supports:

Keystroke    | Action
---------    | ------
Ctrl-A, Home | Move cursor to beginning of line
Ctrl-E, End  | Move cursor to end of line
Ctrl-B, Left | Move cursor one character left
Ctrl-F, Right| Move cursor one character right
Ctrl-Left, Alt-B    | Move cursor to previous word
Ctrl-Right, Alt-F   | Move cursor to next word
Ctrl-D, Del  | (if line is *not* empty) Delete character under cursor
Ctrl-D       | (if line *is* empty) End of File - usually quits application
Ctrl-C       | Reset input (create new empty prompt)
Ctrl-L       | Clear screen (line is unmodified)
Ctrl-T       | Transpose previous character with current character
Ctrl-H, BackSpace | Delete character before cursor
Ctrl-W       | Delete word leading up to cursor
Ctrl-K       | Delete from cursor to end of line
Ctrl-U       | Delete from start of line to cursor
Ctrl-P, Up   | Previous match from history
Ctrl-N, Down | Next match from history
Ctrl-R       | Reverse Search history (Ctrl-S forward, Ctrl-G cancel)
Ctrl-Y       | Paste from Yank buffer (Alt-Y to paste next yank instead)
Tab          | Next completion
Shift-Tab    | (after Tab) Previous completion

Getting started
-----------------

```go
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
)

var (
	history_fn = filepath.Join(os.TempDir(), ".liner_example_history")
	names      = []string{"john", "james", "mary", "nancy"}
)

func main() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range names {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	if f, err := os.Open(history_fn); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	if name, err := line.Prompt("What is your name? "); err == nil {
		log.Print("Got: ", name)
		line.AppendHistory(name)
	} else if err == liner.ErrPromptAborted {
		log.Print("Aborted")
	} else {
		log.Print("Error reading line: ", err)
	}

	if f, err := os.Create(history_fn); err != nil {
		log.Print("Error writing history file: ", err)
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
```

For documentation, see http://godoc.org/github.com/peterh/liner
