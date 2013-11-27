package main

import "time"

// DirUseSource implementations provide a stream of events indicating
// that a particular process has used a given dir
type DirUseSource interface {
	Events() chan DirUseEvent
}

// a DirUseEvent records use of a directory from a particular source
type DirUseEvent struct {
	ProcId int
	Dir    string
}

// currentDirPoller is a DirUseSource which periodically polls
// the current working dir of all processes on the system
// and creates a DirUseEvent each time a current dir change
// is detected for a process
type currentDirPoller struct {
	events chan DirUseEvent
}

func (poller *currentDirPoller) Events() chan DirUseEvent {
	return poller.events
}

func (poller *currentDirPoller) Run() {
	// this currently polls all processes every few
	// seconds. It would be preferable to avoid the polling if possible

	// map of (PID -> previous current dir)
	prevDir := map[int]string{}

	tick := time.Tick(5 * time.Second)
	for _ = range tick {
		procs := scanProcs()
		for _, procInfo := range procs {
			if prevDir[procInfo.Id] != procInfo.CurrentDir {
				prevDir[procInfo.Id] = procInfo.CurrentDir
				poller.events <- DirUseEvent{procInfo.Id, procInfo.CurrentDir}
			}
		}
	}
}

func newCurrentDirPoller() currentDirPoller {
	poller := currentDirPoller{}
	poller.events = make(chan DirUseEvent)
	go poller.Run()
	return poller
}
