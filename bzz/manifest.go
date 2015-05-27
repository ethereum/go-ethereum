package bzz

import (
// "fmt"
)

const (
	manifestType = "application/bzz-manifest+json"
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
		pos = len(entry.Path)
		if len(path) >= pos && path[:pos] == entry.Path {
			dpaLogger.Debugf("Swarm: '%s' matches '%s'.", path, entry.Path)
			return
		}
	}
	entry = nil
	dpaLogger.Debugf("Path '%s' on manifest not found.", path)
	return
}
