package qml

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// ParseResources parses the resources collection serialized in data.
func ParseResources(data []byte) (*Resources, error) {
	if len(data) < 20 || string(data[:4]) != "qres" {
		return nil, fmt.Errorf("invalid resources data")
	}
	r, err := parseResourcesHeader(data[:20], len(data))
	if err != nil {
		return nil, err
	}
	r.bdata = data
	return r, nil
}

// ParseResourcesString parses the resources collection serialized in data.
func ParseResourcesString(data string) (*Resources, error) {
	if len(data) < 20 || data[:4] != "qres" {
		return nil, fmt.Errorf("invalid resources data")
	}
	r, err := parseResourcesHeader([]byte(data[:20]), len(data))
	if err != nil {
		return nil, err
	}
	r.sdata = data
	return r, nil
}

func parseResourcesHeader(h []byte, size int) (*Resources, error) {
	r := &Resources{
		version:    read32(h[4:]),
		treeOffset: read32(h[8:]),
		dataOffset: read32(h[12:]),
		nameOffset: read32(h[16:]),
	}
	if r.version != resVersion {
		return nil, fmt.Errorf("unsupported resources version: %d", r.version)
	}
	// Ideally this would do a full validation, but it's a good start.
	if !(20 <= r.treeOffset && r.treeOffset < size &&
		20 <= r.dataOffset && r.dataOffset < size &&
		20 <= r.nameOffset && r.nameOffset < size) {
		return nil, fmt.Errorf("corrupted resources data")
	}
	return r, nil
}

func read32(b []byte) int {
	return int(uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]))
}

// Resources is a compact representation of a collection of resources
// (images, qml files, etc) that may be loaded by an Engine and referenced
// by QML at "qrc:///some/path", where "some/path" is the path the
// resource was added with.
//
// Resources must be registered with LoadResources to become available.
type Resources struct {
	sdata string
	bdata []byte

	version    int
	treeOffset int
	dataOffset int
	nameOffset int
}

// Bytes returns a binary representation of the resources collection that
// may be parsed back with ParseResources or ParseResourcesString.
func (r *Resources) Bytes() []byte {
	if len(r.sdata) > 0 {
		return []byte(r.sdata)
	}
	return r.bdata
}

// ResourcesPacker builds a Resources collection with provided resources.
type ResourcesPacker struct {
	root resFile
}

// Pack builds a resources collection with all resources previously added.
func (rp *ResourcesPacker) Pack() *Resources {
	rw := newResourcesWriter(rp)
	rw.write()
	return &Resources{
		bdata:      rw.out.Bytes(),
		version:    resVersion,
		dataOffset: rw.dataOffset,
		nameOffset: rw.nameOffset,
		treeOffset: rw.treeOffset,
	}
}

type resFile struct {
	name  string
	sdata string
	bdata []byte

	children resFiles
}

// Add adds a resource with the provided data under "qrc:///"+path.
func (rp *ResourcesPacker) Add(path string, data []byte) {
	file := rp.addFile(path)
	file.bdata = data
}

// AddString adds a resource with the provided data under "qrc:///"+path.
func (rp *ResourcesPacker) AddString(path, data string) {
	file := rp.addFile(path)
	file.sdata = data
}

func (rp *ResourcesPacker) addFile(path string) *resFile {
	file := &rp.root
	names := strings.Split(path, "/")
	if len(names[0]) == 0 {
		names = names[1:]
	}
NextItem:
	for _, name := range names {
		for i := range file.children {
			child := &file.children[i]
			if child.name == name {
				file = child
				continue NextItem
			}
		}
		file.children = append(file.children, resFile{name: name})
		file = &file.children[len(file.children)-1]
	}
	if len(file.children) > 0 || file.sdata != "" || file.bdata != nil {
		panic("cannot add same resources path twice: " + path)
	}
	return file
}

type resWriter struct {
	root *resFile

	treeOffset int
	dataOffset int
	nameOffset int

	treeOffsets map[*resFile]int
	dataOffsets map[*resFile]int
	nameOffsets map[string]int

	pending []*resFile
	out     bytes.Buffer
}

func newResourcesWriter(rp *ResourcesPacker) *resWriter {
	rw := &resWriter{
		root:        &rp.root,
		treeOffsets: make(map[*resFile]int),
		dataOffsets: make(map[*resFile]int),
		nameOffsets: make(map[string]int),
		pending:     make([]*resFile, maxPending(&rp.root)),
	}

	pending := rw.pending
	pending[0] = rw.root
	n := 1
	for n > 0 {
		n--
		file := pending[n]
		sort.Sort(file.children)
		for i := range file.children {
			child := &file.children[i]
			if len(child.children) > 0 {
				pending[n] = child
				n++
			}
		}
	}
	return rw
}

func maxPending(file *resFile) int {
	max := 1
	for i := range file.children {
		if len(file.children) > 0 {
			max += maxPending(&file.children[i])
		}
	}
	return max
}

func (rw *resWriter) write() {
	rw.writeHeader()
	rw.writeDataBlobs()
	rw.writeDataNames()
	rw.writeDataTree()
	rw.finishHeader()
}

func (rw *resWriter) writeHeader() {
	rw.out.WriteString("qres\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
}

func (rw *resWriter) finishHeader() {
	rw.write32at(4, resVersion)
	rw.write32at(8, uint32(rw.treeOffset))
	rw.write32at(12, uint32(rw.dataOffset))
	rw.write32at(16, uint32(rw.nameOffset))
}

func (rw *resWriter) writeDataBlobs() {
	rw.dataOffset = rw.out.Len()
	pending := rw.pending
	pending[0] = rw.root
	n := 1
	for n > 0 {
		n--
		file := pending[n]
		for i := range file.children {
			child := &file.children[i]
			if len(child.children) > 0 {
				pending[n] = child
				n++
			} else {
				rw.dataOffsets[child] = rw.out.Len() - rw.dataOffset
				rw.writeDataBlob(child)
			}
		}
	}
}

func (rw *resWriter) writeDataBlob(file *resFile) {
	if len(file.sdata) > 0 {
		rw.write32(uint32(len(file.sdata)))
		rw.out.WriteString(file.sdata)
	} else {
		rw.write32(uint32(len(file.bdata)))
		rw.out.Write(file.bdata)
	}
}

func (rw *resWriter) writeDataNames() {
	rw.nameOffset = rw.out.Len()
	pending := rw.pending
	pending[0] = rw.root
	n := 1
	for n > 0 {
		n--
		file := pending[n]
		for i := range file.children {
			child := &file.children[i]
			if len(child.children) > 0 {
				pending[n] = child
				n++
			}
			if _, ok := rw.nameOffsets[child.name]; !ok {
				rw.nameOffsets[child.name] = rw.out.Len() - rw.nameOffset
				rw.writeDataName(child.name)
			}
		}
	}
}

func (rw *resWriter) writeDataName(name string) {
	rw.write16(uint16(len(name)))
	rw.write32(qt_hash(name))
	for _, r := range name {
		rw.write16(uint16(r))
	}
}

func (rw *resWriter) writeDataTree() {
	rw.treeOffset = rw.out.Len()

	// Compute first child offset for each parent.
	pending := rw.pending
	pending[0] = rw.root
	n := 1
	offset := 1
	for n > 0 {
		n--
		file := pending[n]
		rw.treeOffsets[file] = offset
		for i := range file.children {
			child := &file.children[i]
			offset++
			if len(child.children) > 0 {
				pending[n] = child
				n++
			}
		}
	}

	// Actually write it out.
	rw.writeDataInfo(rw.root)
	pending[0] = rw.root
	n = 1
	for n > 0 {
		n--
		file := pending[n]
		for i := range file.children {
			child := &file.children[i]
			rw.writeDataInfo(child)
			if len(child.children) > 0 {
				pending[n] = child
				n++
			}
		}
	}
}

func (rw *resWriter) writeDataInfo(file *resFile) {
	rw.write32(uint32(rw.nameOffsets[file.name]))
	if len(file.children) > 0 {
		rw.write16(uint16(resDirectory))
		rw.write32(uint32(len(file.children)))
		rw.write32(uint32(rw.treeOffsets[file]))
	} else {
		rw.write16(uint16(resNone))
		rw.write16(0) // QLocale::AnyCountry
		rw.write16(1) // QLocale::C
		rw.write32(uint32(rw.dataOffsets[file]))
	}
}

const (
	resVersion = 1

	resNone       = 0
	resCompressed = 1
	resDirectory  = 2
)

func (rw *resWriter) write16(v uint16) {
	rw.out.Write([]byte{byte(v >> 8), byte(v)})
}

func (rw *resWriter) write32(v uint32) {
	rw.out.Write([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func (rw *resWriter) write32at(index int, v uint32) {
	b := rw.out.Bytes()
	b[index+0] = byte(v >> 24)
	b[index+1] = byte(v >> 16)
	b[index+2] = byte(v >> 8)
	b[index+3] = byte(v)
}

type resFiles []resFile

func (rf resFiles) Len() int           { return len(rf) }
func (rf resFiles) Less(i, j int) bool { return qt_hash(rf[i].name) < qt_hash(rf[j].name) }
func (rf resFiles) Swap(i, j int)      { rf[i], rf[j] = rf[j], rf[i] }

// qt_hash returns the hash of p as determined by the internal qt_hash function in Qt.
//
// According to the documentation in qhash.cpp this algorithm is used whenever
// the hash may be stored or reused across Qt versions, and must not change.
// The algorithm in qHash (used in QString, etc) is different and may change.
func qt_hash(p string) uint32 {
	var h uint32
	for _, r := range p {
		h = (h << 4) + uint32(r)
		h ^= (h & 0xf0000000) >> 23
		h &= 0x0fffffff
	}
	return h
}
