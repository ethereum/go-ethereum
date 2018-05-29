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
	"encoding/json"
	"github.com/ethereum/go-ethereum/log"
	"github.com/fsnotify/fsnotify"
	"github.com/mohae/deepcopy"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// embrace inserts buf into brackets.
func embrace(buf []byte) []byte {
	b := make([]byte, len(buf)+2)
	b[0] = '['
	copy(b[1:], buf)
	b[len(buf)+1] = ']'

	return b
}

// replaceNewLinesWithCommas replaces the '\n' characters with ',' characters and returns the last replaced position.
func replaceNewLinesWithCommas(buf []byte) int {
	last := -1
	for i := 0; i < len(buf); i++ {
		if buf[i] == '\n' {
			buf[i] = ','
			last = i
		}
	}
	return last
}

// handleLogRequest searches for the log file specified by the timestamp of the request, creates a JSON array out of it
// and sends it to the requesting client.
func (db *Dashboard) handleLogRequest(r *LogsRequest, c *client) {
	files, err := ioutil.ReadDir(db.logdir)
	if err != nil {
		log.Warn("Failed to open logdir", "path", db.logdir, "err", err)
		return
	}
	re := regexp.MustCompile(".log$")
	fileNames := make([]string, len(files))
	n := 0
	for _, f := range files {
		if f.Mode().IsRegular() && re.Match([]byte(f.Name())) {
			fileNames[n] = f.Name()
			n++
		}
	}
	if n < 1 {
		log.Warn("There isn't any log file in the logdir", "path", db.logdir)
		return
	}
	i := sort.Search(n, func(i int) bool {
		return fileNames[i] >= r.Name // Returns the smallest index such as fileNames[i] >= timestamp.
	})

	if i >= n || fileNames[i] != r.Name {
		log.Warn("The requested file isn't in the logdir", "path", filepath.Join(db.logdir, r.Name))
		return
	}

	last := false
	if r.Past {
		if i <= 0 {
			log.Warn("There isn't more log file in the logdir", "path", db.logdir)
			return
		}
		i--
		if i == 0 {
			last = true
		}
	} else {
		if i >= n-1 {
			log.Warn("There isn't more log file in the logdir", "path", db.logdir)
			return
		}
		if i == n-2 {
			// The last file is continuously updated, and its chunks are streamed,
			// so in order to avoid log record duplication on the client side, it is
			// handled differently. Its actual content is always saved in the history.
			db.lock.Lock()
			if db.history.Logs != nil {
				c.msg <- &Message{
					Logs: db.history.Logs,
				}
			}
			db.lock.Unlock()
			return
		}
		i++
	}

	path := filepath.Join(db.logdir, fileNames[i])
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		log.Warn("Failed to open file", "path", path, "err", err)
		return
	}
	defer f.Close()
	var buf []byte
	if buf, err = ioutil.ReadAll(f); err != nil {
		log.Warn("Failed to read file", "path", path, "err", err)
		return
	}
	lastComma := replaceNewLinesWithCommas(buf)
	if lastComma < 0 {
		log.Warn("The file doesn't contain valid logs", "path", path)
		return
	}
	db.lock.Lock()
	c.msg <- &Message{
		Logs: &LogsMessage{
			Old: &LogFile{
				Name: fileNames[i],
				Past: r.Past,
				Last: last,
			},
			Chunk: embrace(buf[:lastComma]),
		},
	}
	db.lock.Unlock()
}

// streamLogs watches the file system, and when the logger writes the new log records into the files, picks them up,
// then makes JSON array out of them and sends them to the clients.
// This could be embedded into collectData, but they shouldn't depend on each other, and also cleaner this way.
func (db *Dashboard) streamLogs() {
	defer db.wg.Done()

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
	re := regexp.MustCompile(".log$")
	var i int
	for i = len(files) - 1; i >= 0 && (!files[i].Mode().IsRegular() || !re.Match([]byte(files[i].Name()))); i-- {
	}
	if i < 0 {
		log.Warn("There isn't any log file in the logdir", "path", db.logdir)
		return
	}
	if opened, err = os.OpenFile(filepath.Join(db.logdir, files[i].Name()), os.O_RDONLY, 0644); err != nil {
		log.Warn("Failed to open file", "name", files[i].Name(), "err", err)
		return
	}
	defer opened.Close() // Close the lastly opened file.
	fi, err := opened.Stat()
	if err != nil {
		log.Warn("Problem with file", "name", opened.Name(), "err", err)
		return
	}
	db.lock.Lock()
	db.history.Logs = &LogsMessage{
		Old: &LogFile{
			Name: fi.Name(),
			Past: false,
			Last: true,
		},
		Chunk: json.RawMessage("[]"),
	}
	db.lock.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Warn("Failed to create fs watcher", "err", err)
		return
	}
	defer watcher.Close()
	err = watcher.Add(db.logdir)
	if err != nil {
		log.Warn("Failed to add logdir to fs watcher", "logdir", db.logdir, "err", err)
		return
	}

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	for {
		select {
		case event := <-watcher.Events:
			// Make sure that new log file was created.
			if event.Op&fsnotify.Create == 0 || !re.Match([]byte(event.Name)) {
				break
			}
			if opened == nil {
				log.Warn("The last log file is not opened")
				return
			}
			// The new log file's name is always greater, because it is created using the actual log record's time.
			if opened.Name() >= event.Name {
				break
			}
			// Read the rest of the previously opened file.
			chunk, err := ioutil.ReadAll(opened)
			if err != nil {
				log.Warn("Failed to read file", "name", opened.Name(), "err", err)
				return
			}
			b := make([]byte, len(buf)+len(chunk))
			copy(b, buf)
			copy(b[len(buf):], chunk)
			buf = b
			opened.Close()

			if last := replaceNewLinesWithCommas(buf); last >= 0 {
				// Send the rest of the previously opened file.
				db.sendToAll(&Message{
					Logs: &LogsMessage{
						Chunk: embrace(buf[:last]),
					},
				})
			}
			if opened, err = os.OpenFile(event.Name, os.O_RDONLY, 0644); err != nil {
				log.Warn("Failed to open file", "name", event.Name, "err", err)
				return
			}
			buf = buf[:0]

			// Change the last file in the history.
			fi, err := opened.Stat()
			if err != nil {
				log.Warn("Problem with file", "name", opened.Name(), "err", err)
				return
			}
			db.lock.Lock()
			db.history.Logs.Old.Name = fi.Name()
			db.history.Logs.Chunk = json.RawMessage("[]")
			db.lock.Unlock()
		case err := <-watcher.Errors:
			if err != nil {
				log.Warn("Fs watcher error", "err", err)
			}
			return
		case errc := <-db.quit:
			errc <- nil
			return
			// Send log updates to the client.
		case <-ticker.C:
			if opened == nil {
				log.Warn("The last log file is not opened")
				return
			}

			// Read the new logs created since the last read.
			chunk, err := ioutil.ReadAll(opened)
			if err != nil {
				log.Warn("Failed to read file", "name", opened.Name(), "err", err)
				return
			}
			b := make([]byte, len(buf)+len(chunk))
			copy(b, buf)
			copy(b[len(buf):], chunk)

			last := replaceNewLinesWithCommas(b)
			if last < 0 {
				break
			}
			// Only keep the invalid part of the buffer, which can be valid after the next read.
			buf = b[last+1:]

			msg := embrace(b[:last])
			var l *LogsMessage
			// Update the history.
			db.lock.Lock()
			if len(db.history.Logs.Chunk) == 2 {
				db.history.Logs.Chunk = msg
				l = deepcopy.Copy(db.history.Logs).(*LogsMessage)
			} else {
				b = make([]byte, len(db.history.Logs.Chunk)+len(msg)-1)
				copy(b, db.history.Logs.Chunk)
				b[len(db.history.Logs.Chunk)-1] = ','
				copy(b[len(db.history.Logs.Chunk):], msg[1:])
				db.history.Logs.Chunk = b
				l = &LogsMessage{Chunk: msg}
			}
			db.lock.Unlock()

			db.sendToAll(&Message{Logs: l})
		}
	}
}
