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

// Data structures used for Fuse filesystem, serving directories and serving files to Fuse driver.

package api

import (
	"io"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"golang.org/x/net/context"
)

type FS struct {
	root *Dir
}

type Dir struct {
	inode       uint64
	name        string
	path        string
	directories []*Dir
	files       []*File
}

type File struct {
	inode    uint64
	name     string
	path     string
	key      storage.Key
	swarmApi *Api
	fileSize uint64
	reader   storage.LazySectionReader
}

// Functions which satisfy the Fuse File System requests
func (filesystem *FS) Root() (fs.Node, error) {
	return filesystem.root, nil
}

func (directory *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = directory.inode
	//TODO: need to get permission as argument
	a.Mode = os.ModeDir | 0500
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getegid())
	return nil
}

func (directory *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if directory.files != nil {
		for _, n := range directory.files {
			if n.name == name {
				return n, nil
			}
		}
	}
	if directory.directories != nil {
		for _, n := range directory.directories {
			if n.name == name {
				return n, nil
			}
		}
	}
	return nil, fuse.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var children []fuse.Dirent
	if d.files != nil {
		for _, file := range d.files {
			children = append(children, fuse.Dirent{Inode: file.inode, Type: fuse.DT_File, Name: file.name})
		}
	}
	if d.directories != nil {
		for _, dir := range d.directories {
			children = append(children, fuse.Dirent{Inode: dir.inode, Type: fuse.DT_Dir, Name: dir.name})
		}
	}
	return children, nil
}

func (file *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = file.inode
	//TODO: need to get permission as argument
	a.Mode = 0500
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getegid())

	reader := file.swarmApi.Retrieve(file.key)
	quitC := make(chan bool)
	size, err := reader.Size(quitC)
	if err != nil {
		log.Warn("Couldnt file size of file %s : %v", file.path, err)
		a.Size = uint64(0)
	}
	a.Size = uint64(size)
	file.fileSize = a.Size
	return nil
}

var _ = fs.HandleReader(&File{})

func (file *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	buf := make([]byte, req.Size)
	reader := file.swarmApi.Retrieve(file.key)
	n, err := reader.ReadAt(buf, req.Offset)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Data = buf[:n]
	return err
}
