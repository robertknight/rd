package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kless/term"
	"github.com/wsxiaoys/terminal/color"
)

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

// returns the number of matches which occur at the start
// of a component in the path
func (match *QueryMatch) ComponentPrefixMatches() int {
	prefixMatches := 0
	for _, offset := range match.MatchOffsets {
		if offset.Start > 0 && match.Dir.Path[offset.Start-1] == '/' {
			prefixMatches++
		}
	}
	return prefixMatches
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

	// timestamp for the last save
	// of dir history to the persistent store
	lastUpdateTime time.Time

	// timestamp for the most recent update
	// to 'recentDirs'
	lastSaveTime time.Time

	// an event source for manual reporting of
	// dir usage via the Push() func
	manualDirSource manualDirSource
}

func NewRecentDirServer() RecentDirServer {
	server := RecentDirServer{}

	procCwdPoller := newCurrentDirPoller()
	server.manualDirSource = newManualDirSource()
	server.sources = []DirUseSource{
		&procCwdPoller,
		&server.manualDirSource,
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (server *RecentDirServer) Push(dir string, reply *bool) error {
	if !fileExists(dir) {
		return errors.New("Dir %s does not exist")
	}

	server.manualDirSource.Push(dir)
	*reply = true
	return nil
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

	return "" // for Go 1.0 compat
}

type QueryMatchSort []QueryMatch

// sort query matches such that list.Less(i,j) is true iff
// list[i] is a better match than list[j]
func (list QueryMatchSort) Less(i, j int) bool {
	prefixMatchesLeft := list[i].ComponentPrefixMatches()
	prefixMatchesRight := list[j].ComponentPrefixMatches()

	if prefixMatchesLeft != prefixMatchesRight {
		return prefixMatchesLeft > prefixMatchesRight
	} else {
		return list[i].Dir.AccessTime.After(list[j].Dir.AccessTime)
	}
}

func (list QueryMatchSort) Len() int {
	return len(list)
}

func (list QueryMatchSort) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
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
			var maxTime time.Time
			for _, index := range indexes {
				aTime := matches[index].Dir.AccessTime
				if aTime.After(maxTime) {
					maxTime = aTime
				}
			}

			result = append(result, QueryMatch{
				Dir:          DirUsage{Path: prefix, AccessTime: maxTime},
				MatchOffsets: matches[indexes[0]].MatchOffsets,
			})
		} else {
			for _, index := range indexes {
				result = append(result, matches[index])
			}
		}
	}

	var sortedResult QueryMatchSort = result
	sort.Sort(sortedResult)

	return []QueryMatch(sortedResult)
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

	parts := strings.Fields(query)
	matchedParts := 0
	for _, part := range parts {
		candidateNorm := strings.ToLower(candidate.Path)
		partNorm := strings.ToLower(part)
		offset := 0
		for {
			subStr := candidateNorm[offset:]
			index := strings.Index(subStr, partNorm)
			if index < 0 {
				break
			}

			match.MatchOffsets = append(match.MatchOffsets, MatchOffset{
				Start:  index + offset,
				Length: len(part),
			})
			matchedParts++
			offset += index + len(part)
		}
	}
	ok = matchedParts == len(parts)

	return
}

func (server *RecentDirServer) recordDirUsage(usage DirUsage) {
	server.recentDirs[usage.Path] = usage
	server.lastUpdateTime = time.Now()
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
			server.recordDirUsage(dirUsage)
		case _ = <-saveTicker:
			if server.lastUpdateTime.After(server.lastSaveTime) {
				server.store.Save(server.recentDirs)
				server.lastSaveTime = time.Now()
			}
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
						if fileExists(match.Dir.Path) {
							match.Dir = usage
							result = append(result, match)
						} else {
							delete(server.recentDirs, match.Dir.Path)
						}
					}
				}
				result = sortGroupMatches(result)
				server.assignResultIds(result)
			}
			query.replyChan <- result
		}
	}
}

type MatchList []MatchOffset

func (m MatchList) Len() int {
	return len(m)
}

func (m MatchList) Less(i, j int) bool {
	return m[i].Start < m[j].Start
}

func (m MatchList) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func highlightMatches(input string, offsets []MatchOffset) string {
	var list MatchList = offsets
	sort.Sort(list)

	output := ""
	offset := 0
	startMarker := color.Colorize("r!")
	endMarker := color.Colorize("|")

	for _, match := range list {
		output += input[offset:match.Start]
		output += startMarker
		output += input[match.Start : match.Start+match.Length]
		output += endMarker
		offset = match.Start + match.Length
	}
	output += input[offset:]

	return output
}

func handleQueryCommand(client *rpc.Client, args []string, useColors bool) {
	query := strings.Join(args, " ")
	reply := []QueryMatch{}
	err := client.Call("RecentDirServer.Query", query, &reply)
	if err != nil {
		fmt.Printf("Failed to query the rd daemon: %v\n", err)
		os.Exit(1)
	}

	if len(reply) == 1 {
		fmt.Println(reply[0].Dir.Path)
	} else {
		for _, match := range reply {
			var highlightedMatch string
			if useColors {
				highlightedMatch = highlightMatches(match.Dir.Path, match.MatchOffsets)
			} else {
				highlightedMatch = match.Dir.Path
			}
			fmt.Printf("  %d: %s\n", match.Id, highlightedMatch)
		}
	}
}

func handlePushCommand(client *rpc.Client, args []string) {
	if len(args) < 1 {
		fmt.Printf("No dir to push specified\n")
		os.Exit(1)
	}

	var reply bool
	err := client.Call("RecentDirServer.Push", args[0], &reply)
	if err != nil {
		fmt.Printf("Failed to query the rd daemon: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	daemonFlag := flag.Bool("daemon", false, "Start rd in daemon mode")
	colorFlag := flag.Bool("color", term.IsTerminal(syscall.Stdout), "Colorize matches in output")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [options] <command> <args...>

Supported commands:
  query <pattern>|<id>
  push <path>

Flags:
`, os.Args[0])
		flag.PrintDefaults()
	}
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
			fmt.Printf("No command given. Use '%s -help' for a list of supported commands\n",
				os.Args[0])
			os.Exit(1)
		}

		client, err := rpc.Dial(connType, connAddr)
		if err != nil {
			fmt.Printf(
				`
Unable to connect to the rd daemon.
 
Use 'rd -daemon' to start it.
This should be set up to run at login.
 
`)
			os.Exit(1)
		}

		modeStr := flag.Arg(0)
		args := []string{}
		if flag.NArg() > 1 {
			args = flag.Args()[1:]
		}

		switch modeStr {
		case "query":
			handleQueryCommand(client, args, *colorFlag)
		case "push":
			handlePushCommand(client, args)
		default:
			fmt.Printf("Unknown command '%s', use '%s -help' for a list of supported commands\n",
				modeStr, os.Args[0])
		}
	}
}
