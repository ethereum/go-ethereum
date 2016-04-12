// Copyright 2016 The go-ethereum Authors
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

package accounts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

type accountsByFile []Account

func (s accountsByFile) Len() int           { return len(s) }
func (s accountsByFile) Less(i, j int) bool { return s[i].File < s[j].File }
func (s accountsByFile) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AmbiguousAddrError is returned when attempting to unlock
// an address for which more than one file exists.
type AmbiguousAddrError struct {
	Addr    common.Address
	Matches []Account
}

func (err *AmbiguousAddrError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.File
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match address (%s)", files)
}

// addrCache is a live index of all accounts in the keystore.
type addrCache struct {
	keydir   string
	watcher  *watcher
	mu       sync.Mutex
	all      accountsByFile
	byAddr   map[common.Address][]Account
	throttle *time.Timer
}

func newAddrCache(keydir string) *addrCache {
	ac := &addrCache{
		keydir: keydir,
		byAddr: make(map[common.Address][]Account),
	}
	ac.watcher = newWatcher(ac)
	return ac
}

func (ac *addrCache) accounts() []Account {
	ac.maybeReload()
	ac.mu.Lock()
	defer ac.mu.Unlock()
	cpy := make([]Account, len(ac.all))
	copy(cpy, ac.all)
	return cpy
}

func (ac *addrCache) hasAddress(addr common.Address) bool {
	ac.maybeReload()
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return len(ac.byAddr[addr]) > 0
}

func (ac *addrCache) add(newAccount Account) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	i := sort.Search(len(ac.all), func(i int) bool { return ac.all[i].File >= newAccount.File })
	if i < len(ac.all) && ac.all[i] == newAccount {
		return
	}
	// newAccount is not in the cache.
	ac.all = append(ac.all, Account{})
	copy(ac.all[i+1:], ac.all[i:])
	ac.all[i] = newAccount
	ac.byAddr[newAccount.Address] = append(ac.byAddr[newAccount.Address], newAccount)
}

// note: removed needs to be unique here (i.e. both File and Address must be set).
func (ac *addrCache) delete(removed Account) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.all = removeAccount(ac.all, removed)
	if ba := removeAccount(ac.byAddr[removed.Address], removed); len(ba) == 0 {
		delete(ac.byAddr, removed.Address)
	} else {
		ac.byAddr[removed.Address] = ba
	}
}

func removeAccount(slice []Account, elem Account) []Account {
	for i := range slice {
		if slice[i] == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// find returns the cached account for address if there is a unique match.
// The exact matching rules are explained by the documentation of Account.
// Callers must hold ac.mu.
func (ac *addrCache) find(a Account) (Account, error) {
	// Limit search to address candidates if possible.
	matches := ac.all
	if (a.Address != common.Address{}) {
		matches = ac.byAddr[a.Address]
	}
	if a.File != "" {
		// If only the basename is specified, complete the path.
		if !strings.ContainsRune(a.File, filepath.Separator) {
			a.File = filepath.Join(ac.keydir, a.File)
		}
		for i := range matches {
			if matches[i].File == a.File {
				return matches[i], nil
			}
		}
		if (a.Address == common.Address{}) {
			return Account{}, ErrNoMatch
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Account{}, ErrNoMatch
	default:
		err := &AmbiguousAddrError{Addr: a.Address, Matches: make([]Account, len(matches))}
		copy(err.Matches, matches)
		return Account{}, err
	}
}

func (ac *addrCache) maybeReload() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if ac.watcher.running {
		return // A watcher is running and will keep the cache up-to-date.
	}
	if ac.throttle == nil {
		ac.throttle = time.NewTimer(0)
	} else {
		select {
		case <-ac.throttle.C:
		default:
			return // The cache was reloaded recently.
		}
	}
	ac.watcher.start()
	ac.reload()
	ac.throttle.Reset(minReloadInterval)
}

func (ac *addrCache) close() {
	ac.mu.Lock()
	ac.watcher.close()
	if ac.throttle != nil {
		ac.throttle.Stop()
	}
	ac.mu.Unlock()
}

// reload caches addresses of existing accounts.
// Callers must hold ac.mu.
func (ac *addrCache) reload() {
	accounts, err := ac.scan()
	if err != nil && glog.V(logger.Debug) {
		glog.Errorf("can't load keys: %v", err)
	}
	ac.all = accounts
	sort.Sort(ac.all)
	for k := range ac.byAddr {
		delete(ac.byAddr, k)
	}
	for _, a := range accounts {
		ac.byAddr[a.Address] = append(ac.byAddr[a.Address], a)
	}
	glog.V(logger.Debug).Infof("reloaded keys, cache has %d accounts", len(ac.all))
}

func (ac *addrCache) scan() ([]Account, error) {
	files, err := ioutil.ReadDir(ac.keydir)
	if err != nil {
		return nil, err
	}

	var (
		buf     = new(bufio.Reader)
		addrs   []Account
		keyJSON struct {
			Address common.Address `json:"address"`
		}
	)
	for _, fi := range files {
		path := filepath.Join(ac.keydir, fi.Name())
		if skipKeyFile(fi) {
			glog.V(logger.Detail).Infof("ignoring file %s", path)
			continue
		}
		fd, err := os.Open(path)
		if err != nil {
			glog.V(logger.Detail).Infoln(err)
			continue
		}
		buf.Reset(fd)
		// Parse the address.
		keyJSON.Address = common.Address{}
		err = json.NewDecoder(buf).Decode(&keyJSON)
		switch {
		case err != nil:
			glog.V(logger.Debug).Infof("can't decode key %s: %v", path, err)
		case (keyJSON.Address == common.Address{}):
			glog.V(logger.Debug).Infof("can't decode key %s: missing or zero address", path)
		default:
			addrs = append(addrs, Account{Address: keyJSON.Address, File: path})
		}
		fd.Close()
	}
	return addrs, err
}

func skipKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}
