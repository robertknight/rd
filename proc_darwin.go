package main

// +build darwin

// #cgo LDFLAGS:-lproc
// #include <libproc.h>
import "C"

import (
	"errors"
	"unsafe"
)

type Proc struct {
	Id         int
	Name       string
	CurrentDir string
}

const PROC_PIDVNODEPATHINFO = 9

type PathInfo struct {
	cdir C.struct_vnode_info_path
	rdir C.struct_vnode_info_path
}

func listPids() []int {
	var buffer [4096]C.int
	pidCount := int(C.proc_listallpids(unsafe.Pointer(&buffer[0]), C.int(len(buffer))))
	result := make([]int, pidCount)
	for i := 0; i < pidCount; i++ {
		result[i] = int(buffer[i])
	}
	return result
}

func getProcCurrentDir(pid int) (string, int) {
	var pathInfo PathInfo
	pathInfoPtr := unsafe.Pointer(&pathInfo)
	err := int(C.proc_pidinfo(C.int(pid), PROC_PIDVNODEPATHINFO, 0, pathInfoPtr,
		C.int(unsafe.Sizeof(pathInfo))))
	return C.GoString(&pathInfo.cdir.vip_path[0]), err
}

func GetProcInfo(id int) (Proc, error) {
	cwd, result := getProcCurrentDir(id)
	if result != 0 {
		return Proc{Id: id, CurrentDir: cwd}, nil
	} else {
		return Proc{Id: id}, errors.New("unable to read pid info")
	}
}

func scanProcs() []Proc {
	pids := listPids()
	procs := []Proc{}
	for _, pid := range pids {
		proc, err := GetProcInfo(pid)
		if err != nil {
			continue
		}
		procs = append(procs, proc)
	}
	return procs
}
