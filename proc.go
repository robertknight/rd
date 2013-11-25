package main

// functions for querying information of running processes.
// Currently only implemented for Linux

import (
	"fmt"
	"os"
	"strconv"
)

type Proc struct {
	Id         int
	Name       string
	CurrentDir string
}

func GetProcInfo(id int) (Proc, error) {
	binLinkPath := fmt.Sprintf("/proc/%d/exe", id)
	cwdLinkPath := fmt.Sprintf("/proc/%d/cwd", id)
	binPath, err := os.Readlink(binLinkPath)
	if err != nil {
		return Proc{}, err
	}

	cwdPath, err := os.Readlink(cwdLinkPath)
	if err != nil {
		return Proc{}, err
	}

	return Proc{Id: id, Name: binPath, CurrentDir: cwdPath}, nil
}

func scanProcs() []Proc {
	procDir, _ := os.Open("/proc")
	files, _ := procDir.Readdir(0)
	procs := []Proc{}
	for _, fileInfo := range files {
		pid, err := strconv.Atoi(fileInfo.Name())
		if err != nil {
			continue
		}
		proc, err := GetProcInfo(pid)
		if err != nil {
			continue
		}
		procs = append(procs, proc)
	}
	return procs
}
