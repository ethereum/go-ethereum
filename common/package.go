package common

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// Manifest object
//
// The manifest object holds all the relevant information supplied with the
// the manifest specified in the package
type Manifest struct {
	Entry         string
	Height, Width int
}

// External package
//
// External package contains the main html file and manifest
type ExtPackage struct {
	EntryHtml string
	Manifest  *Manifest
}

// Read file
//
// Read a given compressed file and returns the read bytes.
// Returns an error otherwise
func ReadFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	content, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// Reads manifest
//
// Reads and returns a manifest object. Returns error otherwise
func ReadManifest(m []byte) (*Manifest, error) {
	var manifest Manifest

	dec := json.NewDecoder(strings.NewReader(string(m)))
	if err := dec.Decode(&manifest); err == io.EOF {
	} else if err != nil {
		return nil, err
	}

	return &manifest, nil
}

// Find file in archive
//
// Returns the index of the given file name if it exists. -1 if file not found
func FindFileInArchive(fn string, files []*zip.File) (index int) {
	index = -1
	// Find the manifest first
	for i, f := range files {
		if f.Name == fn {
			index = i
		}
	}

	return
}

// Open package
//
// Opens a prepared ethereum package
// Reads the manifest file and determines file contents and returns and
// the external package.
func OpenPackage(fn string) (*ExtPackage, error) {
	r, err := zip.OpenReader(fn)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	manifestIndex := FindFileInArchive("manifest.json", r.File)

	if manifestIndex < 0 {
		return nil, fmt.Errorf("No manifest file found in archive")
	}

	f, err := ReadFile(r.File[manifestIndex])
	if err != nil {
		return nil, err
	}

	manifest, err := ReadManifest(f)
	if err != nil {
		return nil, err
	}

	if manifest.Entry == "" {
		return nil, fmt.Errorf("Entry file specified but appears to be empty: %s", manifest.Entry)
	}

	entryIndex := FindFileInArchive(manifest.Entry, r.File)
	if entryIndex < 0 {
		return nil, fmt.Errorf("Entry file not found: '%s'", manifest.Entry)
	}

	f, err = ReadFile(r.File[entryIndex])
	if err != nil {
		return nil, err
	}

	extPackage := &ExtPackage{string(f), manifest}

	return extPackage, nil
}
