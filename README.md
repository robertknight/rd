rd
==

rd is a tool for quickly navigating between recently used directories across different
apps.

rd continuously monitors the current directories of active processes and keeps
track of those which have been recently used.

rd can then query and display a list of recently used dirs which match a given pattern.

A set of scripts are provided for searching and switching to recently used dirs
from the shell, via the `cdr` command.

### Installation

If you have Go installed and your path set up so that $GOPATH/bin is
included in your path:

```
	go get github.com/robertknight/rd
```

See the _Bash Integration_ section below for setting up the `cdr` command for use
in the shell.

### Usage

```
  rd -daemon
     <pattern>
     <id>
```

Running rd with -daemon starts the daemon which monitors process activity
system-wide and keeps track of recently used dirs.

Specifying a <pattern> sends a query to the daemon to search for recently used
dirs which match the pattern. If there is a single match the directory path
is printed. If there are multiple matches, an ID is printed for each match
followed by the path.

Specifying an <id> prints the path output by a previous <pattern> query.

### Bash Integration

rd includes a utility script `rd.bash` which provides a `cdr` command for Bash that
can be used to easily switch to a recently used directory from the shell.

```
  cdr <pattern>
      <id>
```

Invokes `rd` with the given pattern or ID. If the result is a single match,
changes to that directory, otherwise prints a list of matches prefixed by ID.

### Design

rd is a client/server app consisting of a daemon which monitors
usage of recently accessed dirs by different processes and a client
which communicates with the daemon.

The client/server are built into the same binary and communicate
with each other using `go`'s built-in RPC support.

