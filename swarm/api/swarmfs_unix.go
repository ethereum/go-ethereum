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

// +build linux darwin

package api

import (
	"path/filepath"
	"fmt"
	"strings"
	"time"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"bazil.org/fuse"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
	"bazil.org/fuse/fs"
	"sync"
)


var (
	inode  uint64  = 1
	inodeLock   sync.RWMutex
)

// information about every active mount
type MountInfo struct {
	mountPoint     string
	manifestHash   string
	resolvedKey    storage.Key
	rootDir        *Dir
	fuseConnection *fuse.Conn
}

// Inode numbers need to be unique, they are used for caching inside fuse
func NewInode() uint64 {
	inodeLock.Lock()
	defer  inodeLock.Unlock()
	inode += 1
	return inode
}



func (self *SwarmFS) Mount(mhash, mountpoint string) (string, error)  {

	self.activeLock.Lock()
	defer self.activeLock.Unlock()

	noOfActiveMounts := len(self.activeMounts)
	if noOfActiveMounts >= maxFuseMounts {
		err := fmt.Errorf("Max mount count reached. Cannot mount %s ", mountpoint)
		log.Warn(err.Error())
		return err.Error(), err
	}

	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return err.Error(), err
	}

	if _, ok := self.activeMounts[cleanedMountPoint]; ok {
		err := fmt.Errorf("Mountpoint %s already mounted.", cleanedMountPoint)
		log.Warn(err.Error())
		return err.Error(), err
	}

	log.Info(fmt.Sprintf("Attempting to mount %s ", cleanedMountPoint))
	key, _, path, err := self.swarmApi.parseAndResolve(mhash, true)
	if err != nil {
		errStr := fmt.Sprintf("Could not resolve %s : %v", mhash, err)
		log.Warn(errStr)
		return errStr, err
	}

	if len(path) > 0 {
		path += "/"
	}

	quitC := make(chan bool)
	trie, err := loadManifest(self.swarmApi.dpa, key, quitC)
	if err != nil {
		errStr := fmt.Sprintf("fs.Download: loadManifestTrie error: %v", err)
		log.Warn(errStr)
		return errStr, err
	}

	dirTree := map[string]*Dir{}

	rootDir := &Dir{
		inode:       NewInode(),
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
						inode:       NewInode(),
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
			inode:    NewInode(),
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
		errStr := fmt.Sprintf("Mounting %s encountered error: %v", cleanedMountPoint, err)
		log.Warn(errStr)
		return errStr, err
	}

	mounterr := make(chan error, 1)
	go func() {
		log.Info(fmt.Sprintf("Serving %s at %s", mhash, cleanedMountPoint))
		filesys := &FS{root: rootDir}
		if err := fs.Serve(fconn, filesys); err != nil {
			log.Warn(fmt.Sprintf("Could not Serve FS error: %v", err))
		}
	}()

	// Check if the mount process has an error to report.
	select {

	case <-time.After(mountTimeout):
		err := fmt.Errorf("Mounting %s timed out.", cleanedMountPoint)
		log.Warn(err.Error())
		return err.Error(), err

	case err := <-mounterr:
	        errStr := fmt.Sprintf("Mounting %s encountered error: %v", cleanedMountPoint, err)
		log.Warn(errStr)
		return errStr, err

	case <-fconn.Ready:
		log.Debug(fmt.Sprintf("Mounting connection succeeded for : %v", cleanedMountPoint))
	}



	//Assemble and Store the mount information for future use
	mountInformation := &MountInfo{
		mountPoint:     cleanedMountPoint,
		manifestHash:   mhash,
		resolvedKey:    key,
		rootDir:        rootDir,
		fuseConnection: fconn,
	}
	self.activeMounts[cleanedMountPoint] = mountInformation

	succString := fmt.Sprintf("Mounting successful for %s", cleanedMountPoint)
	log.Info(succString)

	return succString, nil
}

func (self *SwarmFS) Unmount(mountpoint string) (string, error)  {

	self.activeLock.Lock()
	defer self.activeLock.Unlock()

	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return err.Error(), err
	}

	// Get the mount information based on the mountpoint argument
	mountInfo := self.activeMounts[cleanedMountPoint]


	if mountInfo == nil || mountInfo.mountPoint != cleanedMountPoint {
		err := fmt.Errorf("Could not find mount information for %s ", cleanedMountPoint)
		log.Warn(err.Error())
		return err.Error(), err
	}

	err = fuse.Unmount(cleanedMountPoint)
	if err != nil {
		//TODO: try forceful unmount if normal unmount fails
		errStr := fmt.Sprintf("UnMount error: %v", err)
		log.Warn(errStr)
		return errStr, err
	}

	mountInfo.fuseConnection.Close()

	//remove the mount information from the active map
	delete(self.activeMounts, cleanedMountPoint)

	succString := fmt.Sprintf("UnMounting %v succeeded", cleanedMountPoint)
	log.Info(succString)
	return succString, nil
}

func (self *SwarmFS) Listmounts() (string, error) {

	self.activeLock.RLock()
	defer self.activeLock.RUnlock()

	var rows []string
	for mp := range self.activeMounts {
		mountInfo := self.activeMounts[mp]
		rows = append(rows, fmt.Sprintf("Swarm Root: %s, Mount Point: %s ", mountInfo.manifestHash, mountInfo.mountPoint))
	}

	return strings.Join(rows, "\n"), nil
}

func (self *SwarmFS) Stop() bool {

	for mp := range self.activeMounts {
		mountInfo := self.activeMounts[mp]
		self.Unmount(mountInfo.mountPoint)
	}

	return true
}
