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

// path must not have any leading slashes
func (self *manifest) getEntry(path string) (matchingEntry *manifestEntry, matchlength int) {
	var pos, depth, maxdepth int
	var entry *manifestEntry
	// path := leadingSlashes.ReplaceAllString(fullpath, "")
	// iterate over entries matching paths to the target
	// due to redundant slashes, it is NOT the longest match but the match with
	// the highest depth is chosen
	// this gives thse matches in case of trailing slashes:
	// "a/" -> "a/" not "a" and "a" matches "a" not "a/" if both exist
	// "a" never matches "a/" but "a/" matches a
	for _, entry = range self.Entries {
		entryPath := leadingSlashes.ReplaceAllString(entry.Path, "")
		pos = len(entryPath)
		depth = len(slashes.Split(entryPath, -1))
		if len(path) >= pos && path[:pos] == entryPath && (depth > maxdepth || depth == maxdepth && pos > matchlength) {
			var hop int
			// hop and chop leading hashes of the continuation
			if len(path) > pos {
				chopped := leadingSlashes.ReplaceAllString(path[pos:], "")
				hop = len(path) - pos - len(chopped)
			}
			// check if pos actually ends a subpath "ab" matches on "" not on "a"
			if hop > 0 || pos == len(path) || pos == 0 || path[pos-1] == '/' {
				matchlength = pos + hop
				maxdepth = depth
				matchingEntry = entry
			}
		}
	}
	if matchingEntry != nil {
		dpaLogger.Debugf("Swarm: '%s' matches '%s'.", path, matchingEntry.Path)
	} else {
		dpaLogger.Debugf("Path '%s' not found on manifest ", path)
	}

	return
}
