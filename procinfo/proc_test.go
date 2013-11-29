package procinfo

import (
	"os"
	"syscall"
	"testing"
)

func TestCurrentDir(t *testing.T) {
	pids := ListPids()

	if len(pids) < 1 {
		t.Error("Unable to list process IDs")
	}

	foundCurrent := false
	for _, pid := range pids {
		proc, err := GetProcInfo(pid)
		if err == nil {
			if len(proc.CurrentDir) == 0 {
				t.Errorf("GetProcInfo returned no error for %d but failed to read current dir", pid)
			}
		}

		if pid == syscall.Getpid() {
			foundCurrent = true
			currentDir, _ := os.Getwd()
			if currentDir != proc.CurrentDir {
				t.Errorf("expected working dir for current process %s, got %s", currentDir, proc.CurrentDir)
			}
		}
	}

	if !foundCurrent {
		t.Errorf("Current process %d not among listed processes", syscall.Getpid())
	}
}
