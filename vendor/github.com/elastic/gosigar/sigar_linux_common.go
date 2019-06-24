// Copyright (c) 2012 VMware, Inc.

// +build freebsd linux

package gosigar

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var system struct {
	ticks uint64
	btime uint64
}

var Procd string

func getLinuxBootTime() {
	// grab system boot time
	readFile(Procd+"/stat", func(line string) bool {
		if strings.HasPrefix(line, "btime") {
			system.btime, _ = strtoull(line[6:])
			return false // stop reading
		}
		return true
	})
}

func (self *LoadAverage) Get() error {
	line, err := ioutil.ReadFile(Procd + "/loadavg")
	if err != nil {
		return nil
	}

	fields := strings.Fields(string(line))

	self.One, _ = strconv.ParseFloat(fields[0], 64)
	self.Five, _ = strconv.ParseFloat(fields[1], 64)
	self.Fifteen, _ = strconv.ParseFloat(fields[2], 64)

	return nil
}

func (self *Swap) Get() error {

	table, err := parseMeminfo()
	if err != nil {
		return err
	}
	self.Total, _ = table["SwapTotal"]
	self.Free, _ = table["SwapFree"]

	self.Used = self.Total - self.Free
	return nil
}

func (self *Cpu) Get() error {
	return readFile(Procd+"/stat", func(line string) bool {
		if len(line) > 4 && line[0:4] == "cpu " {
			parseCpuStat(self, line)
			return false
		}
		return true

	})
}

func (self *CpuList) Get() error {
	capacity := len(self.List)
	if capacity == 0 {
		capacity = 4
	}
	list := make([]Cpu, 0, capacity)

	err := readFile(Procd+"/stat", func(line string) bool {
		if len(line) > 3 && line[0:3] == "cpu" && line[3] != ' ' {
			cpu := Cpu{}
			parseCpuStat(&cpu, line)
			list = append(list, cpu)
		}
		return true
	})

	self.List = list

	return err
}

func (self *FileSystemList) Get() error {
	capacity := len(self.List)
	if capacity == 0 {
		capacity = 10
	}
	fslist := make([]FileSystem, 0, capacity)

	err := readFile(getMountTableFileName(), func(line string) bool {
		fields := strings.Fields(line)

		fs := FileSystem{}
		fs.DevName = fields[0]
		fs.DirName = fields[1]
		fs.SysTypeName = fields[2]
		fs.Options = fields[3]

		fslist = append(fslist, fs)

		return true
	})

	self.List = fslist

	return err
}

func (self *ProcList) Get() error {
	dir, err := os.Open(Procd)
	if err != nil {
		return err
	}
	defer dir.Close()

	const readAllDirnames = -1 // see os.File.Readdirnames doc

	names, err := dir.Readdirnames(readAllDirnames)
	if err != nil {
		return err
	}

	capacity := len(names)
	list := make([]int, 0, capacity)

	for _, name := range names {
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		pid, err := strconv.Atoi(name)
		if err == nil {
			list = append(list, pid)
		}
	}

	self.List = list

	return nil
}

func (self *ProcState) Get(pid int) error {
	data, err := readProcFile(pid, "stat")
	if err != nil {
		return err
	}

	// Extract the comm value with is surrounded by parentheses.
	lIdx := bytes.Index(data, []byte("("))
	rIdx := bytes.LastIndex(data, []byte(")"))
	if lIdx < 0 || rIdx < 0 || lIdx >= rIdx || rIdx+2 >= len(data) {
		return fmt.Errorf("failed to extract comm for pid %d from '%v'", pid, string(data))
	}
	self.Name = string(data[lIdx+1 : rIdx])

	// Extract the rest of the fields that we are interested in.
	fields := bytes.Fields(data[rIdx+2:])
	if len(fields) <= 36 {
		return fmt.Errorf("expected more stat fields for pid %d from '%v'", pid, string(data))
	}

	interests := bytes.Join([][]byte{
		fields[0],  // state
		fields[1],  // ppid
		fields[2],  // pgrp
		fields[4],  // tty_nr
		fields[15], // priority
		fields[16], // nice
		fields[36], // processor (last processor executed on)
	}, []byte(" "))

	var state string
	_, err = fmt.Fscan(bytes.NewBuffer(interests),
		&state,
		&self.Ppid,
		&self.Pgid,
		&self.Tty,
		&self.Priority,
		&self.Nice,
		&self.Processor,
	)
	if err != nil {
		return fmt.Errorf("failed to parse stat fields for pid %d from '%v': %v", pid, string(data), err)
	}
	self.State = RunState(state[0])

	// Read /proc/[pid]/status to get the uid, then lookup uid to get username.
	status, err := getProcStatus(pid)
	if err != nil {
		return fmt.Errorf("failed to read process status for pid %d: %v", pid, err)
	}
	uids, err := getUIDs(status)
	if err != nil {
		return fmt.Errorf("failed to read process status for pid %d: %v", pid, err)
	}
	user, err := user.LookupId(uids[0])
	if err == nil {
		self.Username = user.Username
	} else {
		self.Username = uids[0]
	}

	return nil
}

func (self *ProcMem) Get(pid int) error {
	contents, err := readProcFile(pid, "statm")
	if err != nil {
		return err
	}

	fields := strings.Fields(string(contents))

	size, _ := strtoull(fields[0])
	self.Size = size << 12

	rss, _ := strtoull(fields[1])
	self.Resident = rss << 12

	share, _ := strtoull(fields[2])
	self.Share = share << 12

	contents, err = readProcFile(pid, "stat")
	if err != nil {
		return err
	}

	fields = strings.Fields(string(contents))

	self.MinorFaults, _ = strtoull(fields[10])
	self.MajorFaults, _ = strtoull(fields[12])
	self.PageFaults = self.MinorFaults + self.MajorFaults

	return nil
}

func (self *ProcTime) Get(pid int) error {
	contents, err := readProcFile(pid, "stat")
	if err != nil {
		return err
	}

	fields := strings.Fields(string(contents))

	user, _ := strtoull(fields[13])
	sys, _ := strtoull(fields[14])
	// convert to millis
	self.User = user * (1000 / system.ticks)
	self.Sys = sys * (1000 / system.ticks)
	self.Total = self.User + self.Sys

	// convert to millis
	self.StartTime, _ = strtoull(fields[21])
	self.StartTime /= system.ticks
	self.StartTime += system.btime
	self.StartTime *= 1000

	return nil
}

func (self *ProcArgs) Get(pid int) error {
	contents, err := readProcFile(pid, "cmdline")
	if err != nil {
		return err
	}

	bbuf := bytes.NewBuffer(contents)

	var args []string

	for {
		arg, err := bbuf.ReadBytes(0)
		if err == io.EOF {
			break
		}
		args = append(args, string(chop(arg)))
	}

	self.List = args

	return nil
}

func (self *ProcEnv) Get(pid int) error {
	contents, err := readProcFile(pid, "environ")
	if err != nil {
		return err
	}

	if self.Vars == nil {
		self.Vars = map[string]string{}
	}

	pairs := bytes.Split(contents, []byte{0})
	for _, kv := range pairs {
		parts := bytes.SplitN(kv, []byte{'='}, 2)
		if len(parts) != 2 {
			continue
		}

		key := string(bytes.TrimSpace(parts[0]))
		if key == "" {
			continue
		}

		self.Vars[key] = string(bytes.TrimSpace(parts[1]))
	}

	return nil
}

func (self *ProcExe) Get(pid int) error {
	fields := map[string]*string{
		"exe":  &self.Name,
		"cwd":  &self.Cwd,
		"root": &self.Root,
	}

	for name, field := range fields {
		val, err := os.Readlink(procFileName(pid, name))

		if err != nil {
			return err
		}

		*field = val
	}

	return nil
}

func parseMeminfo() (map[string]uint64, error) {
	table := map[string]uint64{}

	err := readFile(Procd+"/meminfo", func(line string) bool {
		fields := strings.Split(line, ":")

		if len(fields) != 2 {
			return true // skip on errors
		}

		valueUnit := strings.Fields(fields[1])
		value, err := strtoull(valueUnit[0])
		if err != nil {
			return true // skip on errors
		}

		if len(valueUnit) > 1 && valueUnit[1] == "kB" {
			value *= 1024
		}
		table[fields[0]] = value

		return true
	})
	return table, err
}

func readFile(file string, handler func(string) bool) error {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(bytes.NewBuffer(contents))

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if !handler(string(line)) {
			break
		}
	}

	return nil
}

func strtoull(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}

func procFileName(pid int, name string) string {
	return Procd + "/" + strconv.Itoa(pid) + "/" + name
}

func readProcFile(pid int, name string) (content []byte, err error) {
	path := procFileName(pid, name)

	// Panics have been reported when reading proc files, let's recover and
	// report the path if this happens
	// See https://github.com/elastic/beats/issues/6692
	defer func() {
		if r := recover(); r != nil {
			content = nil
			err = fmt.Errorf("recovered panic when reading proc file '%s': %v", path, r)
		}
	}()
	contents, err := ioutil.ReadFile(path)

	if err != nil {
		if perr, ok := err.(*os.PathError); ok {
			if perr.Err == syscall.ENOENT {
				return nil, syscall.ESRCH
			}
		}
	}

	return contents, err
}

// getProcStatus reads /proc/[pid]/status which contains process status
// information in human readable form.
func getProcStatus(pid int) (map[string]string, error) {
	status := make(map[string]string, 42)
	path := filepath.Join(Procd, strconv.Itoa(pid), "status")
	err := readFile(path, func(line string) bool {
		fields := strings.SplitN(line, ":", 2)
		if len(fields) == 2 {
			status[fields[0]] = strings.TrimSpace(fields[1])
		}

		return true
	})
	return status, err
}

// getUIDs reads the "Uid" value from status and splits it into four values --
// real, effective, saved set, and  file system UIDs.
func getUIDs(status map[string]string) ([]string, error) {
	uidLine, ok := status["Uid"]
	if !ok {
		return nil, fmt.Errorf("Uid not found in proc status")
	}

	uidStrs := strings.Fields(uidLine)
	if len(uidStrs) != 4 {
		return nil, fmt.Errorf("Uid line ('%s') did not contain four values", uidLine)
	}

	return uidStrs, nil
}
