#!/usr/bin/env sh

if [ `uname` == 'Darwin' ]
then
	OSX=1
fi

echo "Building 'rd'..."

# Build the 'rd' tool
if [ -z $GOPATH ]
then
	echo "The GOPATH environment variable needs to be set before building"
	exit 1
fi

go get
go build

if [ $? -ne 0 ]
then
	echo "Failed to build rd"
	exit 1
fi

echo "Setting up daemon autostart and shell integration"

RD_BIN_PATH=$PWD/rd

# Autostart the rd daemon at login
if [ $OSX ]
then
	LAUNCHD_AGENT_FILENAME=com.github.robertknight.rd.plist
	sed "s:\$RD_BIN_PATH:$RD_BIN_PATH:" $LAUNCHD_AGENT_FILENAME > ~/Library/LaunchAgents/$LAUNCHD_AGENT_FILENAME
else
	AUTOSTART_DIR=~/.config/autostart
	if [ -d $AUTOSTART_DIR ]
	then
		cp -f $PWD/rd-autostart.desktop $AUTOSTART_DIR/
		echo "Exec=${RD_BIN_PATH} -daemon" >> $AUTOSTART_DIR/rd-autostart.desktop
	fi
fi

# Setup Bash integration
CDR_SCRIPT=$PWD/rd.bash

if [ $OSX ]
then
	SHELL_INIT_FILE=~/.bash_profile
else
	SHELL_INIT_FILE=~/.bashrc
fi

sed -i .bak "/rd.bash/d" $SHELL_INIT_FILE
echo "source \"$CDR_SCRIPT\"" >> $SHELL_INIT_FILE

# Start the rd daemon
if [ $OSX ]
then
	killall rd 2>/dev/null
else
	killall -q rd
fi

nohup rd -daemon 2>/dev/null &

echo "Setup complete. Start a new shell to use the 'cdr' command"
