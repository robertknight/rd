package procinfo

// +build linux

import (
	"fmt"
	"os"
	"strconv"
)

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

	return Proc{Id: id, ExePath: binPath, CurrentDir: cwdPath}, nil
}

func ListPids() []int {
	procDir, _ := os.Open("/proc")
	files, _ := procDir.Readdir(0)
	pids := []int{}
	for _, fileInfo := range files {
		pid, err := strconv.Atoi(fileInfo.Name())
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}
