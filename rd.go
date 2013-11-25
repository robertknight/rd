package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"time"
)

// DirUseSources provide a stream of events indicating
// that a particular process has used a given dir
type DirUseSource interface {
	Events() chan DirUseEvent
}

type DirUseEvent struct {
	ProcId int
	Dir    string
}

type currentDirPoller struct {
	events chan DirUseEvent
}

func (poller *currentDirPoller) Events() chan DirUseEvent {
	return poller.events
}

func (poller *currentDirPoller) Run() {
	tick := time.Tick(2 * time.Second)
	for _ = range tick {
		procs := scanProcs()
		for _, procInfo := range procs {
			poller.events <- DirUseEvent{procInfo.Id, procInfo.CurrentDir}
		}
	}
}

func newCurrentDirPoller() currentDirPoller {
	poller := currentDirPoller{}
	poller.events = make(chan DirUseEvent)
	go poller.Run()
	return poller
}

type Query struct {
	query     string
	replyChan chan []string
}

type DirId struct {
	Id   int
	Path string
	Time time.Time
}

type RecentDirServer struct {
	sources []DirUseSource

	// receives incoming notifications of
	// usage of directories
	eventChan chan DirUseEvent

	// receives incoming queries from rd clients
	// and responds with info about matching directories
	queryChan chan Query

	// map of (id -> path) for recent responses
	// to queries. Used to keep the ID for a given
	// path valid for a period of time
	dirIds []DirId
}

func NewRecentDirServer() RecentDirServer {
	server := RecentDirServer{}

	procCwdPoller := newCurrentDirPoller()
	server.sources = []DirUseSource{
		&procCwdPoller,
	}
	server.eventChan = make(chan DirUseEvent)
	server.queryChan = make(chan Query)

	for _, source := range server.sources {
		go func() {
			for event := range source.Events() {
				server.eventChan <- event
			}
		}()
	}

	return server
}

func (server *RecentDirServer) findDirId(id int) *DirId {
	for _, dir := range server.dirIds {
		if dir.Id == id {
			return &dir
		}
	}
	return nil
}

func (server *RecentDirServer) expireDirIds() {
	const ID_EXPIRY = 10 * time.Second
	newDirIds := []DirId{}
	now := time.Now()
	for _, dir := range server.dirIds {
		if now.Sub(dir.Time) < ID_EXPIRY {
			newDirIds = append(newDirIds, dir)
		}
	}
	server.dirIds = newDirIds
}

func (server *RecentDirServer) Query(queryStr string, reply *[]DirId) error {
	*reply = []DirId{}
	query := Query{
		query:     queryStr,
		replyChan: make(chan []string),
	}
	server.queryChan <- query
	dirs := <-query.replyChan

	server.expireDirIds()
	for _, dir := range dirs {
		var dirId *DirId
		for _, existingDirId := range server.dirIds {
			if existingDirId.Path == dir {
				dirId = &existingDirId
				break
			}
		}
		if dirId == nil {
			newDirId := DirId{Id: len(server.dirIds), Path: dir, Time: time.Now()}
			server.dirIds = append(server.dirIds, newDirId)
			dirId = &newDirId
		}
		*reply = append(*reply, *dirId)
	}

	return nil
}

func (server *RecentDirServer) serve() {
	events := []DirUseEvent{}
	for {
		select {
		case newEvent := <-server.eventChan:
			events = append(events, newEvent)
		case query := <-server.queryChan:
			result := []string{}
			seenDirs := map[string]bool{}

			for _, event := range events {
				if strings.Contains(event.Dir, query.query) &&
					!seenDirs[event.Dir] {
					result = append(result, event.Dir)
					seenDirs[event.Dir] = true
				}
			}
			query.replyChan <- result
		}
	}
}

func main() {
	daemonFlag := flag.Bool("daemon", false, "Start rd in daemon mode")
	tcpPortStr := ":1234"
	flag.Parse()

	if *daemonFlag {
		// server operation
		server := NewRecentDirServer()
		go server.serve()
		rpc.Register(&server)
		rpc.HandleHTTP()
		listener, err := net.Listen("tcp", tcpPortStr)
		if err != nil {
			log.Fatal("Listen error:", err)
		}
		http.Serve(listener, nil)
	} else {
		// client operation
		if flag.NArg() < 1 {
			fmt.Println("No query given")
			return
		}

		client, err := rpc.DialHTTP("tcp", tcpPortStr)
		if err != nil {
			fmt.Printf("Unable to connect to rd daemon: %v\n", err)
			return
		}

		query := flag.Arg(0)
		reply := []DirId{}
		err = client.Call("RecentDirServer.Query", query, &reply)
		if err != nil {
			fmt.Printf("Failed to connect to RD server\n")
			return
		}

		if len(reply) == 1 {
			fmt.Println(reply[0].Path)
		} else {
			for _, dir := range reply {
				fmt.Printf("  %d: %s\n", dir.Id, dir.Path)
			}
		}
	}
}
