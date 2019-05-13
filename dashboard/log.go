// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package dashboard

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mohae/deepcopy"
	"github.com/rjeczalik/notify"
)

var emptyChunk = json.RawMessage("[]")

// prepLogs creates a JSON array from the given log record buffer.
// Returns the prepared array and the position of the last '\n'
// character in the original buffer, or -1 if it doesn't contain any.
func prepLogs(buf []byte) (json.RawMessage, int) {
	b := make(json.RawMessage, 1, len(buf)+1)
	b[0] = '['
	b = append(b, buf...)
	last := -1
	for i := 1; i < len(b); i++ {
		if b[i] == '\n' {
			b[i] = ','
			last = i
		}
	}
	if last < 0 {
		return emptyChunk, -1
	}
	b[last] = ']'
	return b[:last+1], last - 1
}

// handleLogRequest searches for the log file specified by the timestamp of the
// request, creates a JSON array out of it and sends it to the requesting client.
func (db *Dashboard) handleLogRequest(r *LogsRequest, c *client) {
	files, err := ioutil.ReadDir(db.logdir)
	if err != nil {
		log.Warn("Failed to open logdir", "path", db.logdir, "err", err)
		return
	}
	re := regexp.MustCompile(`\.log$`)
	fileNames := make([]string, 0, len(files))
	for _, f := range files {
		if f.Mode().IsRegular() && re.MatchString(f.Name()) {
			fileNames = append(fileNames, f.Name())
		}
	}
	if len(fileNames) < 1 {
		log.Warn("No log files in logdir", "path", db.logdir)
		return
	}
	idx := sort.Search(len(fileNames), func(idx int) bool {
		// Returns the smallest index such as fileNames[idx] >= r.Name,
		// if there is no such index, returns n.
		return fileNames[idx] >= r.Name
	})

	switch {
	case idx < 0:
		return
	case idx == 0 && r.Past:
		return
	case idx >= len(fileNames):
		return
	case r.Past:
		idx--
	case idx == len(fileNames)-1 && fileNames[idx] == r.Name:
		return
	case idx == len(fileNames)-1 || (idx == len(fileNames)-2 && fileNames[idx] == r.Name):
		// The last file is continuously updated, and its chunks are streamed,
		// so in order to avoid log record duplication on the client side, it is
		// handled differently. Its actual content is always saved in the history.
		db.logLock.RLock()
		if db.history.Logs != nil {
			c.msg <- &Message{
				Logs: deepcopy.Copy(db.history.Logs).(*LogsMessage),
			}
		}
		db.logLock.RUnlock()
		return
	case fileNames[idx] == r.Name:
		idx++
	}

	path := filepath.Join(db.logdir, fileNames[idx])
	var buf []byte
	if buf, err = ioutil.ReadFile(path); err != nil {
		log.Warn("Failed to read file", "path", path, "err", err)
		return
	}
	chunk, end := prepLogs(buf)
	if end < 0 {
		log.Warn("The file doesn't contain valid logs", "path", path)
		return
	}
	c.msg <- &Message{
		Logs: &LogsMessage{
			Source: &LogFile{
				Name: fileNames[idx],
				Last: r.Past && idx == 0,
			},
			Chunk: chunk,
		},
	}
}

// streamLogs watches the file system, and when the logger writes
// the new log records into the files, picks them up, then makes
// JSON array out of them and sends them to the clients.
func (db *Dashboard) streamLogs() {
	defer db.wg.Done()
	var (
		err  error
		errc chan error
	)
	defer func() {
		if errc == nil {
			errc = <-db.quit
		}
		errc <- err
	}()

	files, err := ioutil.ReadDir(db.logdir)
	if err != nil {
		log.Warn("Failed to open logdir", "path", db.logdir, "err", err)
		return
	}
	var (
		opened *os.File // File descriptor for the opened active log file.
		buf    []byte   // Contains the recently written log chunks, which are not sent to the clients yet.
	)

	// The log records are always written into the last file in alphabetical order, because of the timestamp.
	re := regexp.MustCompile(`\.log$`)
	i := len(files) - 1
	for i >= 0 && (!files[i].Mode().IsRegular() || !re.MatchString(files[i].Name())) {
		i--
	}
	if i < 0 {
		log.Warn("No log files in logdir", "path", db.logdir)
		return
	}
	if opened, err = os.OpenFile(filepath.Join(db.logdir, files[i].Name()), os.O_RDONLY, 0600); err != nil {
		log.Warn("Failed to open file", "name", files[i].Name(), "err", err)
		return
	}
	defer opened.Close() // Close the lastly opened file.
	fi, err := opened.Stat()
	if err != nil {
		log.Warn("Problem with file", "name", opened.Name(), "err", err)
		return
	}
	db.logLock.Lock()
	db.history.Logs = &LogsMessage{
		Source: &LogFile{
			Name: fi.Name(),
			Last: true,
		},
		Chunk: emptyChunk,
	}
	db.logLock.Unlock()

	watcher := make(chan notify.EventInfo, 10)
	if err := notify.Watch(db.logdir, watcher, notify.Create); err != nil {
		log.Warn("Failed to create file system watcher", "err", err)
		return
	}
	defer notify.Stop(watcher)

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

loop:
	for err == nil || errc == nil {
		select {
		case event := <-watcher:
			// Make sure that new log file was created.
			if !re.Match([]byte(event.Path())) {
				break
			}
			if opened == nil {
				log.Warn("The last log file is not opened")
				break loop
			}
			// The new log file's name is always greater,
			// because it is created using the actual log record's time.
			if opened.Name() >= event.Path() {
				break
			}
			// Read the rest of the previously opened file.
			chunk, err := ioutil.ReadAll(opened)
			if err != nil {
				log.Warn("Failed to read file", "name", opened.Name(), "err", err)
				break loop
			}
			buf = append(buf, chunk...)
			opened.Close()

			if chunk, last := prepLogs(buf); last >= 0 {
				// Send the rest of the previously opened file.
				db.sendToAll(&Message{
					Logs: &LogsMessage{
						Chunk: chunk,
					},
				})
			}
			if opened, err = os.OpenFile(event.Path(), os.O_RDONLY, 0644); err != nil {
				log.Warn("Failed to open file", "name", event.Path(), "err", err)
				break loop
			}
			buf = buf[:0]

			// Change the last file in the history.
			fi, err := opened.Stat()
			if err != nil {
				log.Warn("Problem with file", "name", opened.Name(), "err", err)
				break loop
			}
			db.logLock.Lock()
			db.history.Logs.Source.Name = fi.Name()
			db.history.Logs.Chunk = emptyChunk
			db.logLock.Unlock()
		case <-ticker.C: // Send log updates to the client.
			if opened == nil {
				log.Warn("The last log file is not opened")
				break loop
			}
			// Read the new logs created since the last read.
			chunk, err := ioutil.ReadAll(opened)
			if err != nil {
				log.Warn("Failed to read file", "name", opened.Name(), "err", err)
				break loop
			}
			b := append(buf, chunk...)

			chunk, last := prepLogs(b)
			if last < 0 {
				break
			}
			// Only keep the invalid part of the buffer, which can be valid after the next read.
			buf = b[last+1:]

			var l *LogsMessage
			// Update the history.
			db.logLock.Lock()
			if bytes.Equal(db.history.Logs.Chunk, emptyChunk) {
				db.history.Logs.Chunk = chunk
				l = deepcopy.Copy(db.history.Logs).(*LogsMessage)
			} else {
				b = make([]byte, len(db.history.Logs.Chunk)+len(chunk)-1)
				copy(b, db.history.Logs.Chunk)
				b[len(db.history.Logs.Chunk)-1] = ','
				copy(b[len(db.history.Logs.Chunk):], chunk[1:])
				db.history.Logs.Chunk = b
				l = &LogsMessage{Chunk: chunk}
			}
			db.logLock.Unlock()

			db.sendToAll(&Message{Logs: l})
		case errc = <-db.quit:
			break loop
		}
	}
}
