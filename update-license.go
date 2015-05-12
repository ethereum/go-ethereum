// +build none

/*
This command generates GPL license headers on top of all source files.
You can run it once per month, before cutting a release or just
whenever you feel like it.

	go run update-license.go

The copyright in each file is assigned to any authors for which git
can find commits in the file's history. It will try to follow renames
throughout history. The author names are mapped and deduplicated using
the .mailmap file. You can use .mailmap to set the canonical name and
address for each author. See git-shortlog(1) for an explanation
of the .mailmap format.

Please review the resulting diff to check whether the correct
copyright assignments are performed.
*/
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/template"
    "path/filepath"
)

var (
	// only files with these extensions will be considered
	extensions = []string{".go", ".js", ".qml"}

	// paths with any of these prefixes will be skipped
	skipPrefixes = []string{"Godeps/", "tests/files/", "cmd/mist/assets/ext/", "cmd/mist/assets/muted/"}

	// paths with this prefix are licensed as GPL. all other files are LGPL.
	gplPrefixes = []string{"cmd/"}

	// this regexp must match the entire license comment at the
	// beginning of each file.
	licenseCommentRE = regexp.MustCompile(`(?s)^/\*\s*(Copyright|This file is part of) .*?\*/\n*`)

	// this line is used when git doesn't find any authors for a file
	defaultCopyright = "Copyright (C) 2014 Jeffrey Wilcke <jeffrey@ethereum.org>"
)

// this template generates the license comment.
// its input is an info structure.
var licenseT = template.Must(template.New("").Parse(`/*
	{{.Copyrights}}

	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU {{.License}} as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU {{.License}} for more details.

	You should have received a copy of the GNU {{.License}}
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/

`))

type info struct {
	file    string
	mode    os.FileMode
	authors map[string][]string // map keys are authors, values are years
	gpl     bool
}

func (i info) Copyrights() string {
	var lines []string
	for name, years := range i.authors {
		lines = append(lines, "Copyright (C) "+strings.Join(years, ", ")+" "+name)
	}
	if len(lines) == 0 {
		lines = []string{defaultCopyright}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n\t")
}

func (i info) License() string {
	if i.gpl {
		return "General Public License"
	} else {
		return "Lesser General Public License"
	}
}

func (i info) ShortLicense() string {
	if i.gpl {
		return "GPL"
	} else {
		return "LGPL"
	}
}

func (i *info) addAuthorYear(name, year string) {
	for _, y := range i.authors[name] {
		if y == year {
			return
		}
	}
	i.authors[name] = append(i.authors[name], year)
	sort.Strings(i.authors[name])
}

func main() {
	files := make(chan string)
	infos := make(chan *info)
	wg := new(sync.WaitGroup)

	go getFiles(files)
	for i := runtime.NumCPU(); i >= 0; i-- {
		// getting file info is slow and needs to be parallel
		wg.Add(1)
		go getInfo(files, infos, wg)
	}
	go func() { wg.Wait(); close(infos) }()
	writeLicenses(infos)
}

func getFiles(out chan<- string) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
	err := doLines(cmd, func(line string) {
		for _, p := range skipPrefixes {
			if strings.HasPrefix(line, p) {
				return
			}
		}
		ext := filepath.Ext(line)
		for _, wantExt := range extensions {
			if ext == wantExt {
				goto send
			}
		}
		return

	send:
		out <- line
	})
	if err != nil {
		fmt.Println("error getting files:", err)
	}
	close(out)
}

func getInfo(files <-chan string, out chan<- *info, wg *sync.WaitGroup) {
	for file := range files {
		stat, err := os.Lstat(file)
		if err != nil {
			fmt.Printf("ERROR %s: %v\n", file, err)
			continue
		}
		if !stat.Mode().IsRegular() {
			continue
		}
		info, err := fileInfo(file)
		if err != nil {
			fmt.Printf("ERROR %s: %v\n", file, err)
			continue
		}
		info.mode = stat.Mode()
		out <- info
	}
	wg.Done()
}

func fileInfo(file string) (*info, error) {
	info := &info{file: file, authors: make(map[string][]string)}
	for _, p := range gplPrefixes {
		if strings.HasPrefix(file, p) {
			info.gpl = true
			break
		}
	}
	cmd := exec.Command("git", "log", "--follow", "--find-copies", "--pretty=format:%ai | %aN <%aE>", "--", file)
	err := doLines(cmd, func(line string) {
		sep := strings.IndexByte(line, '|')
		year, name := line[:4], line[sep+2:]
		info.addAuthorYear(name, year)
	})
	return info, err
}

func writeLicenses(infos <-chan *info) {
	buf := new(bytes.Buffer)
	for info := range infos {
		content, err := ioutil.ReadFile(info.file)
		if err != nil {
			fmt.Printf("ERROR: couldn't read %s: %v\n", info.file, err)
			continue
		}

		// construct new file content
		buf.Reset()
		licenseT.Execute(buf, info)
		if m := licenseCommentRE.FindIndex(content); m != nil && m[0] == 0 {
			buf.Write(content[m[1]:])
		} else {
			buf.Write(content)
		}

		if !bytes.Equal(content, buf.Bytes()) {
			fmt.Println("writing", info.ShortLicense(), info.file)
			if err := ioutil.WriteFile(info.file, buf.Bytes(), info.mode); err != nil {
				fmt.Printf("ERROR: couldn't write %s: %v", info.file, err)
			}
		}
	}
}

func doLines(cmd *exec.Cmd, f func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s := bufio.NewScanner(stdout)
	for s.Scan() {
		f(s.Text())
	}
	if s.Err() != nil {
		return s.Err()
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%v (for %s)", err, strings.Join(cmd.Args, " "))
	}
	return nil
}
