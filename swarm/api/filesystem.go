package api

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const maxParallelFiles = 5

type FileSystem struct {
	api *Api
}

func NewFileSystem(api *Api) *FileSystem {
	return &FileSystem{api}
}

// Upload replicates a local directory as a manifest file and uploads it
// using dpa store
// TODO: localpath should point to a manifest
func (self *FileSystem) Upload(lpath, index string) (string, error) {
	var list []*manifestTrieEntry
	localpath, err := filepath.Abs(filepath.Clean(lpath))
	if err != nil {
		return "", err
	}

	f, err := os.Open(localpath)
	if err != nil {
		return "", err
	}
	stat, err := f.Stat()
	if err != nil {
		return "", err
	}

	var start int
	if stat.IsDir() {
		start = len(localpath)
		glog.V(logger.Debug).Infof("[BZZ] uploading '%s'", localpath)
		err = filepath.Walk(localpath, func(path string, info os.FileInfo, err error) error {
			if (err == nil) && !info.IsDir() {
				//fmt.Printf("lp %s  path %s\n", localpath, path)
				if len(path) <= start {
					return fmt.Errorf("Path is too short")
				}
				if path[:start] != localpath {
					return fmt.Errorf("Path prefix of '%s' does not match localpath '%s'", path, localpath)
				}
				entry := &manifestTrieEntry{
					Path: path,
				}
				list = append(list, entry)
			}
			return err
		})
		if err != nil {
			return "", err
		}
	} else {
		dir := filepath.Dir(localpath)
		start = len(dir)
		if len(localpath) <= start {
			return "", fmt.Errorf("Path is too short")
		}
		if localpath[:start] != dir {
			return "", fmt.Errorf("Path prefix of '%s' does not match dir '%s'", localpath, dir)
		}
		entry := &manifestTrieEntry{
			Path: localpath,
		}
		list = append(list, entry)
	}

	cnt := len(list)
	errors := make([]error, cnt)
	done := make(chan bool, maxParallelFiles)
	dcnt := 0

	for i, entry := range list {
		if i >= dcnt+maxParallelFiles {
			<-done
			dcnt++
		}
		go func(i int, entry *manifestTrieEntry, done chan bool) {
			f, err := os.Open(entry.Path)
			if err == nil {
				stat, _ := f.Stat()
				sr := io.NewSectionReader(f, 0, stat.Size())
				wg := &sync.WaitGroup{}
				var hash storage.Key
				hash, err = self.api.dpa.Store(sr, wg)
				if hash != nil {
					list[i].Hash = hash.String()
				}
				wg.Wait()
				if err == nil {
					first512 := make([]byte, 512)
					fread, _ := sr.ReadAt(first512, 0)
					if fread > 0 {
						mimeType := http.DetectContentType(first512[:fread])
						if filepath.Ext(entry.Path) == ".css" {
							mimeType = "text/css"
						}
						list[i].ContentType = mimeType
					}
				}
				f.Close()
			}
			errors[i] = err
			done <- true
		}(i, entry, done)
	}
	for dcnt < cnt {
		<-done
		dcnt++
	}

	trie := &manifestTrie{
		dpa: self.api.dpa,
	}
	for i, entry := range list {
		if errors[i] != nil {
			return "", errors[i]
		}
		entry.Path = RegularSlashes(entry.Path[start:])
		if entry.Path == index {
			ientry := &manifestTrieEntry{
				Path:        "",
				Hash:        entry.Hash,
				ContentType: entry.ContentType,
			}
			trie.addEntry(ientry)
		}
		trie.addEntry(entry)
	}

	err2 := trie.recalcAndStore()
	var hs string
	if err2 == nil {
		hs = trie.hash.String()
	}
	return hs, err2
}

// Download replicates the manifest path structure on the local filesystem
// under localpath
func (self *FileSystem) Download(bzzpath, localpath string) error {
	lpath, err := filepath.Abs(filepath.Clean(localpath))
	if err != nil {
		return err
	}
	err = os.MkdirAll(lpath, os.ModePerm)
	if err != nil {
		return err
	}

	//resolving host and port
	key, _, path, err := self.api.parseAndResolve(bzzpath, true)
	if err != nil {
		return err
	}
	// if len(path) > 0 {
	// 	path += "/"
	// }

	trie, err := loadManifest(self.api.dpa, key)
	if err != nil {
		glog.V(logger.Warn).Infof("[BZZ] fs.Download: loadManifestTrie error: %v", err)
		return err
	}

	type downloadListEntry struct {
		key  storage.Key
		path string
	}

	var list []*downloadListEntry
	var mde, mderr error

	prevPath := lpath
	err = trie.listWithPrefix(path, func(entry *manifestTrieEntry, suffix string) { // TODO: paralellize
		glog.V(logger.Detail).Infof("[BZZ] fs.Download: %#v", entry)

		key := common.Hex2Bytes(entry.Hash)
		path := lpath + "/" + suffix
		dir := filepath.Dir(path)
		if dir != prevPath {
			mde = os.MkdirAll(dir, os.ModePerm)
			if mde != nil {
				mderr = mde
			}
			prevPath = dir
		}
		if (mde == nil) && (path != dir+"/") {
			list = append(list, &downloadListEntry{key: key, path: path})
		}
	})
	if err == nil {
		err = mderr
	}

	cnt := len(list)
	errors := make([]error, cnt)
	done := make(chan bool, maxParallelFiles)
	dcnt := 0

	for i, entry := range list {
		if i >= dcnt+maxParallelFiles {
			<-done
			dcnt++
		}
		go func(i int, entry *downloadListEntry, done chan bool) {
			f, err := os.Create(entry.path) // TODO: path separators
			if err == nil {
				reader := self.api.dpa.Retrieve(entry.key)
				writer := bufio.NewWriter(f)
				_, err = io.CopyN(writer, reader, reader.Size()) // TODO: handle errors
				err2 := writer.Flush()
				if err == nil {
					err = err2
				}
				err2 = f.Close()
				if err == nil {
					err = err2
				}
			}

			errors[i] = err
			done <- true
		}(i, entry, done)
	}
	for dcnt < cnt {
		<-done
		dcnt++
	}

	if err != nil {
		return err
	}
	for i, _ := range list {
		if errors[i] != nil {
			return errors[i]
		}
	}
	return err
}
