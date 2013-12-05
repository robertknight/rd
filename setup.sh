#!/usr/bin/env sh

RD_BIN_PATH=$PWD/rd
SCRIPT_DIR=$PWD/integration/

if [ `uname` = 'Darwin' ]
then
	OSX=1
fi

echo "Building 'rd'..."

# Build the 'rd' tool
if [ -z $GOPATH ]
then
	export GOPATH=/tmp/gocode
fi

go get
go build -o $RD_BIN_PATH

if [ $? -ne 0 ]
then
	echo "Failed to build rd"
	exit 1
fi

echo "Setting up daemon autostart and shell integration"

# Autostart the rd daemon at login
if [ $OSX ]
then
	LAUNCHD_AGENT_FILENAME=com.github.robertknight.rd.plist
	sed "s:\$RD_BIN_PATH:$RD_BIN_PATH:" $SCRIPT_DIR/$LAUNCHD_AGENT_FILENAME > ~/Library/LaunchAgents/$LAUNCHD_AGENT_FILENAME
else
	AUTOSTART_DIR=~/.config/autostart
	if [ -d $AUTOSTART_DIR ]
	then
		cp -f $SCRIPT_DIR/rd-autostart.desktop $AUTOSTART_DIR/
		echo "Exec=${RD_BIN_PATH} -daemon" >> $AUTOSTART_DIR/rd-autostart.desktop
	fi
fi

# Setup Bash shell integration
CDR_SCRIPT=$SCRIPT_DIR/rd.bash

if [ $OSX ]
then
	BASH_INIT_FILE=~/.bash_profile
else
	BASH_INIT_FILE=~/.bashrc
fi

BASH_SOURCE_CMD="source \"$CDR_SCRIPT\""
if [ -e $BASH_INIT_FILE ] ; then
	sed -i.bak "/rd.bash/d" $BASH_INIT_FILE
fi
echo $BASH_SOURCE_CMD >> $BASH_INIT_FILE

# Setup Fish Shell integration
FISH_CONFIG_DIR=~/.config/fish
if [ -d $FISH_CONFIG_DIR ]
then
	FISH_FUNCTIONS_DIR=$FISH_CONFIG_DIR/functions
	mkdir -p $FISH_FUNCTIONS_DIR
	cp $SCRIPT_DIR/cdr.fish $FISH_FUNCTIONS_DIR
fi

# Restart the rd daemon
$RD_BIN_PATH stop
$RD_BIN_PATH query

echo "Setup complete. The 'cdr' command will be available in new shells"
