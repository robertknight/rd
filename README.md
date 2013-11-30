rd
==

rd is a tool for quickly navigating between recently used directories across different
apps.

rd continuously monitors the current directories of active processes and maintains
a database of those which are frequently used. This allows it to automatically learn which dirs you
have used frequently/recently in different apps - a little like the awesomebar / omnibox in
Firefox/Chrome.

The 'cdr' shell command can be used to jump to a recently used dir matching a given pattern.
The database can also be queried or edited directly using the 'rd' command.

### Installation

First, install Go 1.0 or later. Binary packages are available from http://golang.org
On Linux, you can also use:

```
	sudo apt-get install golang-go
```

Then download and build rd and setup shell integration using:

```
	git clone http://github.com/robertknight/rd
	cd rd
	./setup.sh
```

This will:
 1. Build the 'rd' tool
 2. Add an entry to your shell's init file (~/.bashrc for Bash, ~/.config/fish/fish.config for fish)
 3. Setup autostart of the daemon on login
  * On Linux this adds an entry to ~/.config/autostart
  * On Mac this adds an entry to ~/Library/LaunchAgents

You'll need to open a new shell to make the 'cdr' command available.

### Usage

Once rd is installed, it will monitor the current directories of processes on your system.
You can then use the 'cdr' command to jump to a recently used dir.

```
  cdr <pattern>
```

Will jump to the recently used dir containing '<pattern>', if there are multiple matches
a numbered list will be printed and you'll be prompted to choose a match.

### The 'rd' command

The 'rd' command can also be invoked directly to query the database and add new entries.
This can be used to integrate rd into other apps besides the shell.

```
  rd query <pattern>|<id>
     push <path>
     list
     stop
```

Specifying a <pattern> sends a query to the daemon to search for recently used
dirs which match the pattern. If there is a single match the directory path
is printed. If there are multiple matches, an ID is printed for each match
followed by the path.

Specifying an <id> prints the path output by the previous <pattern> query.

The 'push' command can be used to explicitly add a dir to rd's history rather than
waiting for it to discover the dir when it becomes a process's current dir.

The 'stop' command stops the daemon if it is running. It will be automatically started
if necessary when 'rd' is next invoked.

'rd' maintains its history in the `~/.rd-history` file.

### Design

rd is a client/server app consisting of a daemon which monitors
usage of recently accessed dirs by different processes and a client
which communicates with the daemon.

The client/server are built into the same binary and communicate
with each other using `go`'s built-in RPC support.

### See Also

 * [autojump](https://github.com/joelthelion/autojump) is a cd replacement that learns frequently used dirs
 * This [Unix StackExchange](http://unix.stackexchange.com/questions/31161/quick-directory-navigation-in-the-terminal) thread has a collection of tips for faster directory navigation
