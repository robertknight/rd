#!/usr/bin/env sh

if [ `uname` = 'Darwin' ]
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

# Setup Bash shell integration
CDR_SCRIPT=$PWD/rd.bash

if [ $OSX ]
then
	SHELL_INIT_FILE=~/.bash_profile
else
	SHELL_INIT_FILE=~/.bashrc
fi

SHELL_SOURCE_CMD="source \"$CDR_SCRIPT\""
sed -i.bak "/rd.bash/d" $SHELL_INIT_FILE
echo $SHELL_SOURCE_CMD >> $SHELL_INIT_FILE

# Setup Fish Shell integration
FISH_CONFIG_DIR=~/.config/fish
if [ -d $FISH_CONFIG_DIR ]
then
	FISH_INIT_FILE=$FISH_CONFIG_DIR/config.fish
	sed -i.bak "/rd.fish/d" $FISH_INIT_FILE
	echo ". $PWD/rd.fish" >> $FISH_INIT_FILE
fi

# Restart the rd daemon
$RD_BIN_PATH stop
$RD_BIN_PATH query

echo "Setup complete. The 'cdr' command will be available in new shells"
