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

First, install Go 1.0 or later and set GOPATH. Binary packages are available from http://golang.org
On Linux, you can use:

```
	sudo apt-get install golang-go
	export GOPATH=$HOME/go
```

To build rd and install the helper scripts:

```
	./setup.sh
```

This will:
 1. Build the 'rd' tool
 2. Add an entry to ~/.bashrc to load bash integration on login
 3. Add an entry to ~/.config/autostart to start the 'rd' daemon on next login

### Usage

```
  rd -daemon
     query <pattern>|<id>
     push <path>
     list
```

Running rd with -daemon starts the daemon which monitors process activity
system-wide and keeps track of recently used dirs.

Specifying a <pattern> sends a query to the daemon to search for recently used
dirs which match the pattern. If there is a single match the directory path
is printed. If there are multiple matches, an ID is printed for each match
followed by the path.

Specifying an <id> prints the path output by a previous <pattern> query.

The 'push' command can be used to explicitly add a dir to rd's history rather than
waiting for it to discover the dir when it becomes a process's current dir.

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

