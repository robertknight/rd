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
	replyChan chan []QueryMatch
}

type DirUsage struct {
	Path       string
	AccessTime time.Time
}

type MatchOffset struct {
	Start  int
	Length int
}

type QueryMatch struct {
	Id           int
	Dir          DirUsage
	MatchOffsets []MatchOffset
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

	// map of (path -> ID) from the last pattern
	// query. The list of IDs is reset after
	// each new incoming pattern query
	pathIds map[string]int

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
	server.pathIds = make(map[string]int)

	for _, source := range server.sources {
		go func() {
			for event := range source.Events() {
				server.eventChan <- event
			}
		}()
	}

	return server
}

func (server *RecentDirServer) findPathById(findId int) string {
	for path, id := range server.pathIds {
		if id == findId {
			return path
		}
	}
	return ""
}

func (server *RecentDirServer) Query(queryStr string, reply *[]QueryMatch) error {
	query := Query{
		query:     queryStr,
		replyChan: make(chan []QueryMatch),
	}
	server.queryChan <- query
	*reply = <-query.replyChan
	return nil
}

func (server *RecentDirServer) assignId(path string) int {
	for existingPath, id := range server.pathIds {
		if existingPath == path {
			return id
		}
	}
	id := len(server.pathIds) + 1
	server.pathIds[path] = id
	return id
}

// returns the components in a dir path up to and including
// the last part which is included in a match
func matchedPrefix(path string, matches []MatchOffset) string {
	matchEnd := 0
	for _, offset := range matches {
		end := offset.Start + offset.Length
		if end > matchEnd {
			matchEnd = end
		}
	}

	pathSepOffset := strings.Index(path[matchEnd:], "/")
	if pathSepOffset > -1 {
		return path[0 : matchEnd+pathSepOffset]
	} else {
		return path
	}
}

func sortGroupMatches(matches []QueryMatch) []QueryMatch {
	// if there are multiple matches which share a common prefix
	// where all of the matches occur in the common prefix then
	// return only the common prefix
	prefixMatches := map[string][]int{}

	for index, match := range matches {
		prefix := matchedPrefix(match.Dir.Path, match.MatchOffsets)
		if prefixMatches[prefix] == nil {
			prefixMatches[prefix] = []int{}
		}
		prefixMatches[prefix] = append(prefixMatches[prefix], index)
	}

	result := []QueryMatch{}
	for prefix, indexes := range prefixMatches {
		if len(indexes) > 2 {
			result = append(result, QueryMatch{
				Dir:          DirUsage{Path: prefix},
				MatchOffsets: matches[indexes[0]].MatchOffsets,
			})
		} else {
			for _, index := range indexes {
				result = append(result, matches[index])
			}
		}
	}

	return result
}

func (server *RecentDirServer) assignResultIds(matches []QueryMatch) {
	server.pathIds = map[string]int{}
	for index, match := range matches {
		matches[index].Id = server.assignId(match.Dir.Path)
	}
}

func (server *RecentDirServer) queryMatch(query string, candidate DirUsage) (match QueryMatch, ok bool) {
	match.Dir = candidate
	match.MatchOffsets = []MatchOffset{}

	parts := strings.Split(query, " ")
	matchedParts := 0
	for _, part := range parts {
		index := strings.Index(strings.ToLower(candidate.Path),
			strings.ToLower(part))
		if index >= 0 {
			match.MatchOffsets = append(match.MatchOffsets, MatchOffset{
				Start:  index,
				Length: len(part),
			})
			matchedParts++
		} else {
			break
		}
	}
	ok = matchedParts == len(parts)

	return
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
			result := []QueryMatch{}

			queryId, err := strconv.Atoi(query.query)
			if err == nil {
				for path, id := range server.pathIds {
					if id == queryId {
						result = append(result, QueryMatch{
							Dir: DirUsage{Path: path},
						})
						break
					}
				}
			} else {
				for _, usage := range server.recentDirs {
					match, ok := server.queryMatch(query.query, usage)
					if ok {
						match.Dir = usage
						result = append(result, match)
					}
				}
				result = sortGroupMatches(result)
				server.assignResultIds(result)
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
		reply := []QueryMatch{}
		err = client.Call("RecentDirServer.Query", query, &reply)
		if err != nil {
			fmt.Printf("Failed to connect to RD server: %v\n", err)
			return
		}

		if len(reply) == 1 {
			fmt.Println(reply[0].Dir.Path)
		} else {
			for _, match := range reply {
				fmt.Printf("  %d: %s\n", match.Id, match.Dir.Path)
			}
		}
	}
}
