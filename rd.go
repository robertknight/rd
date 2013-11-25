package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
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
	tick := time.Tick(5 * time.Second)
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
	replyChan chan []DirUsage
}

type DirUsage struct {
	Id         int
	Path       string
	AccessTime time.Time
}

type RecentDirServer struct {
	sources []DirUseSource

	// receives incoming notifications of
	// usage of directories
	eventChan chan DirUseEvent

	// receives incoming queries from rd clients
	// and responds with info about matching directories
	queryChan chan Query

	// map of (path -> usage info) for
	// recently used dirs
	recentDirs map[string]DirUsage

	store recentDirStore
}

func NewRecentDirServer() RecentDirServer {
	server := RecentDirServer{}

	procCwdPoller := newCurrentDirPoller()
	server.sources = []DirUseSource{
		&procCwdPoller,
	}
	server.eventChan = make(chan DirUseEvent)
	server.queryChan = make(chan Query)
	server.store = NewRecentDirStore(os.ExpandEnv("$HOME/.rd-history"))
	server.recentDirs = server.store.Load()

	for _, source := range server.sources {
		go func() {
			for event := range source.Events() {
				server.eventChan <- event
			}
		}()
	}

	return server
}

func (server *RecentDirServer) findDirUsage(id int) *DirUsage {
	for _, usage := range server.recentDirs {
		if usage.Id == id {
			return &usage
		}
	}
	return nil
}

func (server *RecentDirServer) Query(queryStr string, reply *[]DirUsage) error {
	query := Query{
		query:     queryStr,
		replyChan: make(chan []DirUsage),
	}
	server.queryChan <- query
	*reply = <-query.replyChan
	return nil
}

func (server *RecentDirServer) maxId() int {
	maxId := 0
	for _, usage := range server.recentDirs {
		if usage.Id > maxId {
			maxId = usage.Id
		}
	}
	return maxId
}

func (server *RecentDirServer) queryMatch(query string, candidate DirUsage) bool {
	id, err := strconv.Atoi(query)
	if err == nil {
		return id != 0 && candidate.Id == id
	} else {
		parts := strings.Split(query, " ")
		matchedParts := 0
		for _, part := range parts {
			if strings.Contains(strings.ToLower(candidate.Path),
				strings.ToLower(part)) {
				matchedParts++
			} else {
				break
			}
		}
		return matchedParts == len(parts)
	}
}

func (server *RecentDirServer) serve() {
	saveTicker := time.Tick(5 * time.Second)
	for {
		select {
		case newEvent := <-server.eventChan:
			dirUsage, ok := server.recentDirs[newEvent.Dir]
			if !ok {
				dirUsage = DirUsage{
					Path: newEvent.Dir,
				}
				log.Printf("recording new dir %s (total: %d)", newEvent.Dir, len(server.recentDirs)+1)
			}
			dirUsage.AccessTime = time.Now()
			server.recentDirs[newEvent.Dir] = dirUsage
		case _ = <-saveTicker:
			server.store.Save(server.recentDirs)
		case query := <-server.queryChan:
			result := []DirUsage{}
			for dir, usage := range server.recentDirs {
				if server.queryMatch(query.query, usage) {
					if usage.Id == 0 {
						usage.Id = server.maxId() + 1
						server.recentDirs[dir] = usage
					}
					result = append(result, usage)
				}
			}
			query.replyChan <- result
		}
	}
}

func main() {
	daemonFlag := flag.Bool("daemon", false, "Start rd in daemon mode")
	flag.Parse()

	connType := "unix"
	connAddr := os.ExpandEnv("$HOME/.rd.sock")

	if *daemonFlag {
		err := os.Remove(connAddr)
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("Unable to remove socket - %v", err)
		}

		// server operation
		server := NewRecentDirServer()
		go server.serve()
		rpcServer := rpc.NewServer()
		rpcServer.Register(&server)
		listener, err := net.Listen(connType, connAddr)
		if err != nil {
			log.Fatal("Listen error:", err)
		}
		rpcServer.Accept(listener)
	} else {
		// client operation
		if flag.NArg() < 1 {
			fmt.Println("No query given")
			return
		}

		client, err := rpc.Dial(connType, connAddr)
		if err != nil {
			fmt.Printf("Unable to connect to rd daemon: %v\n", err)
			return
		}

		query := strings.Join(flag.Args(), " ")
		reply := []DirUsage{}
		err = client.Call("RecentDirServer.Query", query, &reply)
		if err != nil {
			fmt.Printf("Failed to connect to RD server: %v\n", err)
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
