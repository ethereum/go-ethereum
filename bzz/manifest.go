package bzz

import (
	// "fmt"
	"regexp"
)

const (
	manifestType = "application/bzz-manifest+json"
)

var (
	leadingSlashes = regexp.MustCompile("^/+")
)

type manifest struct {
	Entries []*manifestEntry `json:"entries"`
}

type manifestEntry struct {
	Path        string `json:"path"`
	Hash        string `json:"hash"`
	ContentType string `json:"contentType"`
	Status      int    `json:"status"`
}

func (self *manifest) getEntry(path string) (entry *manifestEntry, pos int) {
	for _, entry = range self.Entries {
		entryPath := leadingSlashes.ReplaceAllString(entry.Path, "")
		pos = len(entryPath)
		if len(path) >= pos && path[:pos] == entryPath {
			var n int
			if len(path) > pos {
				chopped := leadingSlashes.ReplaceAllString(path[pos:], "")
				n = len(path) - pos - len(chopped)
			}
			if n > 0 || pos == 0 || path[pos-1] == '/' {
				pos += n
				dpaLogger.Debugf("Swarm: '%s' matches '%s'.", path, entry.Path)
				return
			}
		}
	}
	entry = nil
	dpaLogger.Debugf("Path '%s' on manifest not found.", path)
	return
}
