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

package fuse

import (
	"errors"
	"io"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"golang.org/x/net/context"
)

const (
	MaxAppendFileSize = 10485760 // 10Mb
)

var (
	errInvalidOffset           = errors.New("Invalid offset during write")
	errFileSizeMaxLimixReached = errors.New("File size exceeded max limit")
)

var (
	_ fs.Node         = (*SwarmFile)(nil)
	_ fs.HandleReader = (*SwarmFile)(nil)
	_ fs.HandleWriter = (*SwarmFile)(nil)
)

type SwarmFile struct {
	inode    uint64
	name     string
	path     string
	addr     storage.Address
	fileSize int64
	reader   storage.LazySectionReader

	mountInfo *MountInfo
	lock      *sync.RWMutex
}

func NewSwarmFile(path, fname string, minfo *MountInfo) *SwarmFile {
	newFile := &SwarmFile{
		inode:    NewInode(),
		name:     fname,
		path:     path,
		addr:     nil,
		fileSize: -1, // -1 means , file already exists in swarm and you need to just get the size from swarm
		reader:   nil,

		mountInfo: minfo,
		lock:      &sync.RWMutex{},
	}
	return newFile
}

func (sf *SwarmFile) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("swarmfs Attr", "path", sf.path)
	sf.lock.Lock()
	defer sf.lock.Unlock()
	a.Inode = sf.inode
	//TODO: need to get permission as argument
	a.Mode = 0700
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getegid())

	if sf.fileSize == -1 {
		reader, _ := sf.mountInfo.swarmApi.Retrieve(ctx, sf.addr)
		quitC := make(chan bool)
		size, err := reader.Size(ctx, quitC)
		if err != nil {
			log.Error("Couldnt get size of file %s : %v", sf.path, err)
			return err
		}
		sf.fileSize = size
		log.Trace("swarmfs Attr", "size", size)
		close(quitC)
	}
	a.Size = uint64(sf.fileSize)
	return nil
}

func (sf *SwarmFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	log.Debug("swarmfs Read", "path", sf.path, "req.String", req.String())
	sf.lock.RLock()
	defer sf.lock.RUnlock()
	if sf.reader == nil {
		sf.reader, _ = sf.mountInfo.swarmApi.Retrieve(ctx, sf.addr)
	}
	buf := make([]byte, req.Size)
	n, err := sf.reader.ReadAt(buf, req.Offset)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Data = buf[:n]
	sf.reader = nil

	return err
}

func (sf *SwarmFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Debug("swarmfs Write", "path", sf.path, "req.String", req.String())
	if sf.fileSize == 0 && req.Offset == 0 {
		// A new file is created
		err := addFileToSwarm(sf, req.Data, len(req.Data))
		if err != nil {
			return err
		}
		resp.Size = len(req.Data)
	} else if req.Offset <= sf.fileSize {
		totalSize := sf.fileSize + int64(len(req.Data))
		if totalSize > MaxAppendFileSize {
			log.Warn("swarmfs Append file size reached (%v) : (%v)", sf.fileSize, len(req.Data))
			return errFileSizeMaxLimixReached
		}

		err := appendToExistingFileInSwarm(sf, req.Data, req.Offset, int64(len(req.Data)))
		if err != nil {
			return err
		}
		resp.Size = len(req.Data)
	} else {
		log.Warn("swarmfs Invalid write request size(%v) : off(%v)", sf.fileSize, req.Offset)
		return errInvalidOffset
	}
	return nil
}
