package sourcemap // import "gopkg.in/sourcemap.v1"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/sourcemap.v1/base64vlq"
)

type Consumer struct {
	sourceRootURL *url.URL
	smap          *sourceMap
	mappings      []mapping
}

func Parse(mapURL string, b []byte) (*Consumer, error) {
	smap := new(sourceMap)
	err := json.Unmarshal(b, smap)
	if err != nil {
		return nil, err
	}

	if smap.Version != 3 {
		return nil, errors.New("sourcemap: only 3rd version is supported")
	}

	var sourceRootURL *url.URL
	if smap.SourceRoot != "" {
		u, err := url.Parse(smap.SourceRoot)
		if err != nil {
			return nil, err
		}
		if u.IsAbs() {
			sourceRootURL = u
		}
	} else if mapURL != "" {
		u, err := url.Parse(mapURL)
		if err != nil {
			return nil, err
		}
		if u.IsAbs() {
			u.Path = path.Dir(u.Path)
			sourceRootURL = u
		}
	}

	mappings, err := parseMappings(smap.Mappings)
	if err != nil {
		return nil, err
	}
	// Free memory.
	smap.Mappings = ""

	return &Consumer{
		sourceRootURL: sourceRootURL,
		smap:          smap,
		mappings:      mappings,
	}, nil
}

func (c *Consumer) File() string {
	return c.smap.File
}

func (c *Consumer) Source(genLine, genCol int) (source, name string, line, col int, ok bool) {
	i := sort.Search(len(c.mappings), func(i int) bool {
		m := &c.mappings[i]
		if m.genLine == genLine {
			return m.genCol >= genCol
		}
		return m.genLine >= genLine
	})

	// Mapping not found.
	if i == len(c.mappings) {
		return
	}

	match := &c.mappings[i]

	// Fuzzy match.
	if match.genCol > genCol && i > 0 {
		match = &c.mappings[i-1]
	}

	if match.sourcesInd >= 0 {
		source = c.absSource(c.smap.Sources[match.sourcesInd])
	}
	if match.namesInd >= 0 {
		iv := c.smap.Names[match.namesInd]
		switch v := iv.(type) {
		case string:
			name = v
		case float64:
			name = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			name = fmt.Sprint(iv)
		}
	}
	line = match.sourceLine
	col = match.sourceCol
	ok = true
	return
}

func (c *Consumer) absSource(source string) string {
	if path.IsAbs(source) {
		return source
	}

	if u, err := url.Parse(source); err == nil && u.IsAbs() {
		return source
	}

	if c.sourceRootURL != nil {
		u := *c.sourceRootURL
		u.Path = path.Join(c.sourceRootURL.Path, source)
		return u.String()
	}

	if c.smap.SourceRoot != "" {
		return path.Join(c.smap.SourceRoot, source)
	}

	return source
}

func (c *Consumer) SourceName(genLine, genCol int, genName string) (name string, ok bool) {
	ind := sort.Search(len(c.mappings), func(i int) bool {
		m := c.mappings[i]
		if m.genLine == genLine {
			return m.genCol >= genCol
		}
		return m.genLine >= genLine
	})

	// Mapping not found.
	if ind == len(c.mappings) {
		return "", false
	}

	for i := ind; i >= 0; i-- {
		m := c.mappings[i]
		if m.namesInd == -1 {
			continue
		}
		if c.smap.Names[m.namesInd] == "" {

		}
	}

	return
}

type fn func() (fn, error)

type sourceMap struct {
	Version    int           `json:"version"`
	File       string        `json:"file"`
	SourceRoot string        `json:"sourceRoot"`
	Sources    []string      `json:"sources"`
	Names      []interface{} `json:"names"`
	Mappings   string        `json:"mappings"`
}

type mapping struct {
	genLine    int
	genCol     int
	sourcesInd int
	sourceLine int
	sourceCol  int
	namesInd   int
}

type mappings struct {
	rd  *strings.Reader
	dec *base64vlq.Decoder

	genLine    int
	genCol     int
	sourcesInd int
	sourceLine int
	sourceCol  int
	namesInd   int

	value  mapping
	values []mapping
}

func parseMappings(s string) ([]mapping, error) {
	rd := strings.NewReader(s)
	m := &mappings{
		rd:  rd,
		dec: base64vlq.NewDecoder(rd),

		genLine:    1,
		sourceLine: 1,
	}
	m.zeroValue()
	err := m.parse()
	if err != nil {
		return nil, err
	}
	return m.values, nil
}

func (m *mappings) parse() error {
	next := m.parseGenCol
	for {
		c, err := m.rd.ReadByte()
		if err == io.EOF {
			m.pushValue()
			return nil
		} else if err != nil {
			return err
		}

		switch c {
		case ',':
			m.pushValue()
			next = m.parseGenCol
		case ';':
			m.pushValue()

			m.genLine++
			m.genCol = 0

			next = m.parseGenCol
		default:
			m.rd.UnreadByte()

			var err error
			next, err = next()
			if err != nil {
				return err
			}
		}
	}
}

func (m *mappings) parseGenCol() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.genCol += n
	m.value.genCol = m.genCol
	return m.parseSourcesInd, nil
}

func (m *mappings) parseSourcesInd() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourcesInd += n
	m.value.sourcesInd = m.sourcesInd
	return m.parseSourceLine, nil
}

func (m *mappings) parseSourceLine() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourceLine += n
	m.value.sourceLine = m.sourceLine
	return m.parseSourceCol, nil
}

func (m *mappings) parseSourceCol() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.sourceCol += n
	m.value.sourceCol = m.sourceCol
	return m.parseNamesInd, nil
}

func (m *mappings) parseNamesInd() (fn, error) {
	n, err := m.dec.Decode()
	if err != nil {
		return nil, err
	}
	m.namesInd += n
	m.value.namesInd = m.namesInd
	return m.parseGenCol, nil
}

func (m *mappings) zeroValue() {
	m.value = mapping{
		genLine:    m.genLine,
		genCol:     0,
		sourcesInd: -1,
		sourceLine: 0,
		sourceCol:  0,
		namesInd:   -1,
	}
}

func (m *mappings) pushValue() {
	m.values = append(m.values, m.value)
	m.zeroValue()
}
