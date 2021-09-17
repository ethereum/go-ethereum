// Copyright (c) 2016 Jasper Lievisse Adriaanse <j@jasper.la>.

// +build openbsd

package gosigar

/*
#include <sys/param.h>
#include <sys/types.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <sys/sched.h>
#include <sys/swap.h>
#include <stdlib.h>
#include <unistd.h>
*/
import "C"

//import "github.com/davecgh/go-spew/spew"

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

type Uvmexp struct {
	pagesize           uint32
	pagemask           uint32
	pageshift          uint32
	npages             uint32
	free               uint32
	active             uint32
	inactive           uint32
	paging             uint32
	wired              uint32
	zeropages          uint32
	reserve_pagedaemon uint32
	reserve_kernel     uint32
	anonpages          uint32
	vnodepages         uint32
	vtextpages         uint32
	freemin            uint32
	freetarg           uint32
	inactarg           uint32
	wiredmax           uint32
	anonmin            uint32
	vtextmin           uint32
	vnodemin           uint32
	anonminpct         uint32
	vtextmi            uint32
	npct               uint32
	vnodeminpct        uint32
	nswapdev           uint32
	swpages            uint32
	swpginuse          uint32
	swpgonly           uint32
	nswget             uint32
	nanon              uint32
	nanonneeded        uint32
	nfreeanon          uint32
	faults             uint32
	traps              uint32
	intrs              uint32
	swtch              uint32
	softs              uint32
	syscalls           uint32
	pageins            uint32
	obsolete_swapins   uint32
	obsolete_swapouts  uint32
	pgswapin           uint32
	pgswapout          uint32
	forks              uint32
	forks_ppwait       uint32
	forks_sharevm      uint32
	pga_zerohit        uint32
	pga_zeromiss       uint32
	zeroaborts         uint32
	fltnoram           uint32
	fltnoanon          uint32
	fltpgwait          uint32
	fltpgrele          uint32
	fltrelck           uint32
	fltrelckok         uint32
	fltanget           uint32
	fltanretry         uint32
	fltamcopy          uint32
	fltnamap           uint32
	fltnomap           uint32
	fltlget            uint32
	fltget             uint32
	flt_anon           uint32
	flt_acow           uint32
	flt_obj            uint32
	flt_prcopy         uint32
	flt_przero         uint32
	pdwoke             uint32
	pdrevs             uint32
	pdswout            uint32
	pdfreed            uint32
	pdscans            uint32
	pdanscan           uint32
	pdobscan           uint32
	pdreact            uint32
	pdbusy             uint32
	pdpageouts         uint32
	pdpending          uint32
	pddeact            uint32
	pdreanon           uint32
	pdrevnode          uint32
	pdrevtext          uint32
	fpswtch            uint32
	kmapent            uint32
}

type Bcachestats struct {
	numbufs        uint64
	numbufpages    uint64
	numdirtypages  uint64
	numcleanpages  uint64
	pendingwrites  uint64
	pendingreads   uint64
	numwrites      uint64
	numreads       uint64
	cachehits      uint64
	busymapped     uint64
	dmapages       uint64
	highpages      uint64
	delwribufs     uint64
	kvaslots       uint64
	kvaslots_avail uint64
}

type Swapent struct {
	se_dev      C.dev_t
	se_flags    int32
	se_nblks    int32
	se_inuse    int32
	se_priority int32
	sw_path     []byte
}

func (self *FileSystemList) Get() error {
	num, err := syscall.Getfsstat(nil, C.MNT_NOWAIT)
	if err != nil {
		return err
	}

	buf := make([]syscall.Statfs_t, num)

	_, err = syscall.Getfsstat(buf, C.MNT_NOWAIT)
	if err != nil {
		return err
	}

	fslist := make([]FileSystem, 0, num)

	for i := 0; i < num; i++ {
		fs := FileSystem{}

		fs.DirName = bytePtrToString(&buf[i].F_mntonname[0])
		fs.DevName = bytePtrToString(&buf[i].F_mntfromname[0])
		fs.SysTypeName = bytePtrToString(&buf[i].F_fstypename[0])

		fslist = append(fslist, fs)
	}

	self.List = fslist

	return err
}

func (self *FileSystemUsage) Get(path string) error {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return err
	}

	self.Total = uint64(stat.F_blocks) * uint64(stat.F_bsize)
	self.Free = uint64(stat.F_bfree) * uint64(stat.F_bsize)
	self.Avail = uint64(stat.F_bavail) * uint64(stat.F_bsize)
	self.Used = self.Total - self.Free
	self.Files = stat.F_files
	self.FreeFiles = stat.F_ffree

	return nil
}

func (self *FDUsage) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *LoadAverage) Get() error {
	avg := []C.double{0, 0, 0}

	C.getloadavg(&avg[0], C.int(len(avg)))

	self.One = float64(avg[0])
	self.Five = float64(avg[1])
	self.Fifteen = float64(avg[2])

	return nil
}

func (self *Uptime) Get() error {
	tv := syscall.Timeval{}
	mib := [2]int32{C.CTL_KERN, C.KERN_BOOTTIME}

	n := uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, 0, uintptr(unsafe.Pointer(&n)), 0, 0)

	if errno != 0 || n == 0 {
		return nil
	}

	// Now perform the actual sysctl(3) call, storing the result in tv
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, uintptr(unsafe.Pointer(&tv)), uintptr(unsafe.Pointer(&n)), 0, 0)

	if errno != 0 || n == 0 {
		return nil
	}

	self.Length = time.Since(time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)).Seconds()

	return nil
}

func (self *Mem) Get() error {
	n := uintptr(0)

	var uvmexp Uvmexp
	mib := [2]int32{C.CTL_VM, C.VM_UVMEXP}
	n = uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, uintptr(unsafe.Pointer(&uvmexp)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	var bcachestats Bcachestats
	mib3 := [3]int32{C.CTL_VFS, C.VFS_GENERIC, C.VFS_BCACHESTAT}
	n = uintptr(0)
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib3[0])), 3, 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib3[0])), 3, uintptr(unsafe.Pointer(&bcachestats)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	self.Total = uint64(uvmexp.npages) << uvmexp.pageshift
	self.Used = uint64(uvmexp.npages-uvmexp.free) << uvmexp.pageshift
	self.Free = uint64(uvmexp.free) << uvmexp.pageshift

	self.ActualFree = self.Free + (uint64(bcachestats.numbufpages) << uvmexp.pageshift)
	self.ActualUsed = self.Used - (uint64(bcachestats.numbufpages) << uvmexp.pageshift)

	return nil
}

func (self *Swap) Get() error {
	nswap := C.swapctl(C.SWAP_NSWAP, unsafe.Pointer(uintptr(0)), 0)

	// If there are no swap devices, nothing to do here.
	if nswap == 0 {
		return nil
	}

	swdev := make([]Swapent, nswap)

	rnswap := C.swapctl(C.SWAP_STATS, unsafe.Pointer(&swdev[0]), nswap)
	if rnswap == 0 {
		return nil
	}

	for i := 0; i < int(nswap); i++ {
		if swdev[i].se_flags&C.SWF_ENABLE == 2 {
			self.Used = self.Used + uint64(swdev[i].se_inuse/(1024/C.DEV_BSIZE))
			self.Total = self.Total + uint64(swdev[i].se_nblks/(1024/C.DEV_BSIZE))
		}
	}

	self.Free = self.Total - self.Used

	return nil
}

func (self *Cpu) Get() error {
	load := [C.CPUSTATES]C.long{C.CP_USER, C.CP_NICE, C.CP_SYS, C.CP_INTR, C.CP_IDLE}

	mib := [2]int32{C.CTL_KERN, C.KERN_CPTIME}
	n := uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, uintptr(unsafe.Pointer(&load)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	self.User = uint64(load[0])
	self.Nice = uint64(load[1])
	self.Sys = uint64(load[2])
	self.Irq = uint64(load[3])
	self.Idle = uint64(load[4])

	return nil
}

func (self *CpuList) Get() error {
	mib := [2]int32{C.CTL_HW, C.HW_NCPU}
	var ncpu int

	n := uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, 0, uintptr(unsafe.Pointer(&n)), 0, 0)

	if errno != 0 || n == 0 {
		return nil
	}

	// Now perform the actual sysctl(3) call, storing the result in ncpu
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, uintptr(unsafe.Pointer(&ncpu)), uintptr(unsafe.Pointer(&n)), 0, 0)

	if errno != 0 || n == 0 {
		return nil
	}

	load := [C.CPUSTATES]C.long{C.CP_USER, C.CP_NICE, C.CP_SYS, C.CP_INTR, C.CP_IDLE}

	self.List = make([]Cpu, ncpu)
	for curcpu := range self.List {
		sysctlCptime(ncpu, curcpu, &load)
		fillCpu(&self.List[curcpu], load)
	}

	return nil
}

func (self *ProcList) Get() error {
	return nil
}

func (self *ProcArgs) Get(pid int) error {
	return nil
}

func (self *ProcEnv) Get(pid int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *ProcState) Get(pid int) error {
	return nil
}

func (self *ProcMem) Get(pid int) error {
	return nil
}

func (self *ProcTime) Get(pid int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *ProcExe) Get(pid int) error {
	return nil
}

func (self *ProcFDUsage) Get(pid int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func fillCpu(cpu *Cpu, load [C.CPUSTATES]C.long) {
	cpu.User = uint64(load[0])
	cpu.Nice = uint64(load[1])
	cpu.Sys = uint64(load[2])
	cpu.Irq = uint64(load[3])
	cpu.Idle = uint64(load[4])
}

func sysctlCptime(ncpu int, curcpu int, load *[C.CPUSTATES]C.long) error {
	var mib []int32

	// Use the correct mib based on the number of CPUs and fill out the
	// current CPU number in case of SMP. (0 indexed cf. self.List)
	if ncpu == 0 {
		mib = []int32{C.CTL_KERN, C.KERN_CPTIME}
	} else {
		mib = []int32{C.CTL_KERN, C.KERN_CPTIME2, int32(curcpu)}
	}

	len := len(mib)

	n := uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), uintptr(len), 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), uintptr(len), uintptr(unsafe.Pointer(load)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	return nil
}
