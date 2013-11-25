rd
==

rd is a tool for recording and searching recently accessed directories.
Currently only supports Linux.

### Installation

If you have Go installed and your path set up so that $GOPATH/bin is
included in your path:

```
	go get github.com/robertknight/rd
```

### Usage

```
	1. rd -daemon
	2. rd <pattern>

The first mode starts the daemon which keeps track of recently used dirs
by different processes.

The second mode sends a query to the daemon to search for recently
used dirs which match a given pattern.

Output:
	<id> <path>
```

### Design

rd is a client/server app consisting of a daemon which monitors
usage of recently accessed dirs by different processes and a client
which communicates with the daemon.

The client/server are built into the same binary and communicate
with each other using `go`'s built-in RPC support.

