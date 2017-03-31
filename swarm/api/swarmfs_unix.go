// Copyright 2017 The go-ethereum Authors
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

// +build linux darwin freebsd

package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	inode     uint64 = 1
	inodeLock sync.RWMutex
)

var (
	errEmptyMountPoint = errors.New("need non-empty mount point")
	errMaxMountCount   = errors.New("max FUSE mount count reached")
	errMountTimeout    = errors.New("mount timeout")
)

func isFUSEUnsupportedError(err error) bool {
	if perr, ok := err.(*os.PathError); ok {
		return perr.Op == "open" && perr.Path == "/dev/fuse"
	}
	return err == fuse.ErrOSXFUSENotFound
}

// MountInfo contains information about every active mount
type MountInfo struct {
	MountPoint     string
	ManifestHash   string
	resolvedKey    storage.Key
	rootDir        *Dir
	fuseConnection *fuse.Conn
}

// newInode creates a new inode number.
// Inode numbers need to be unique, they are used for caching inside fuse
func newInode() uint64 {
	inodeLock.Lock()
	defer inodeLock.Unlock()
	inode += 1
	return inode
}

func (self *SwarmFS) Mount(mhash, mountpoint string) (*MountInfo, error) {
	if mountpoint == "" {
		return nil, errEmptyMountPoint
	}
	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return nil, err
	}

	self.activeLock.Lock()
	defer self.activeLock.Unlock()

	noOfActiveMounts := len(self.activeMounts)
	if noOfActiveMounts >= maxFuseMounts {
		return nil, errMaxMountCount
	}

	if _, ok := self.activeMounts[cleanedMountPoint]; ok {
		return nil, fmt.Errorf("%s is already mounted", cleanedMountPoint)
	}

	key, _, path, err := self.swarmApi.parseAndResolve(mhash, true)
	if err != nil {
		return nil, fmt.Errorf("can't resolve %q: %v", mhash, err)
	}

	if len(path) > 0 {
		path += "/"
	}

	quitC := make(chan bool)
	trie, err := loadManifest(self.swarmApi.dpa, key, quitC)
	if err != nil {
		return nil, fmt.Errorf("can't load manifest %v: %v", key.String(), err)
	}

	dirTree := map[string]*Dir{}

	rootDir := &Dir{
		inode:       newInode(),
		name:        "root",
		directories: nil,
		files:       nil,
	}
	dirTree["root"] = rootDir

	err = trie.listWithPrefix(path, quitC, func(entry *manifestTrieEntry, suffix string) {
		key = common.Hex2Bytes(entry.Hash)
		fullpath := "/" + suffix
		basepath := filepath.Dir(fullpath)
		filename := filepath.Base(fullpath)

		parentDir := rootDir
		dirUntilNow := ""
		paths := strings.Split(basepath, "/")
		for i := range paths {
			if paths[i] != "" {
				thisDir := paths[i]
				dirUntilNow = dirUntilNow + "/" + thisDir

				if _, ok := dirTree[dirUntilNow]; !ok {
					dirTree[dirUntilNow] = &Dir{
						inode:       newInode(),
						name:        thisDir,
						path:        dirUntilNow,
						directories: nil,
						files:       nil,
					}
					parentDir.directories = append(parentDir.directories, dirTree[dirUntilNow])
					parentDir = dirTree[dirUntilNow]

				} else {
					parentDir = dirTree[dirUntilNow]
				}

			}
		}
		thisFile := &File{
			inode:    newInode(),
			name:     filename,
			path:     fullpath,
			key:      key,
			swarmApi: self.swarmApi,
		}
		parentDir.files = append(parentDir.files, thisFile)
	})

	fconn, err := fuse.Mount(cleanedMountPoint, fuse.FSName("swarmfs"), fuse.VolumeName(mhash))
	if err != nil {
		fuse.Unmount(cleanedMountPoint)
		log.Warn("Error mounting swarm manifest", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err
	}

	mounterr := make(chan error, 1)
	go func() {
		filesys := &FS{root: rootDir}
		if err := fs.Serve(fconn, filesys); err != nil {
			mounterr <- err
		}
	}()

	// Check if the mount process has an error to report.
	select {
	case <-time.After(mountTimeout):
		fuse.Unmount(cleanedMountPoint)
		return nil, errMountTimeout

	case err := <-mounterr:
		log.Warn("Error serving swarm FUSE FS", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err

	case <-fconn.Ready:
		log.Info("Now serving swarm FUSE FS", "manifest", mhash, "mountpoint", cleanedMountPoint)
	}

	// Assemble and Store the mount information for future use
	mi := &MountInfo{
		MountPoint:     cleanedMountPoint,
		ManifestHash:   mhash,
		resolvedKey:    key,
		rootDir:        rootDir,
		fuseConnection: fconn,
	}
	self.activeMounts[cleanedMountPoint] = mi
	return mi, nil
}

func (self *SwarmFS) Unmount(mountpoint string) (bool, error) {
	self.activeLock.Lock()
	defer self.activeLock.Unlock()

	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return false, err
	}

	mountInfo := self.activeMounts[cleanedMountPoint]
	if mountInfo == nil || mountInfo.MountPoint != cleanedMountPoint {
		return false, fmt.Errorf("%s is not mounted", cleanedMountPoint)
	}
	err = fuse.Unmount(cleanedMountPoint)
	if err != nil {
		// TODO(jmozah): try forceful unmount if normal unmount fails
		return false, err
	}

	// remove the mount information from the active map
	mountInfo.fuseConnection.Close()
	delete(self.activeMounts, cleanedMountPoint)
	return true, nil
}

func (self *SwarmFS) Listmounts() []*MountInfo {
	self.activeLock.RLock()
	defer self.activeLock.RUnlock()

	rows := make([]*MountInfo, 0, len(self.activeMounts))
	for _, mi := range self.activeMounts {
		rows = append(rows, mi)
	}
	return rows
}

func (self *SwarmFS) Stop() bool {
	for mp := range self.activeMounts {
		mountInfo := self.activeMounts[mp]
		self.Unmount(mountInfo.MountPoint)
	}
	return true
}
