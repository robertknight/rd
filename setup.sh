#!/usr/bin/env sh

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
AUTOSTART_DIR=~/.config/autostart
cp -f $PWD/rd-autostart.desktop $AUTOSTART_DIR/
echo "Exec=${RD_BIN_PATH} -daemon" >> $AUTOSTART_DIR/rd-autostart.desktop

# Setup Bash integration
CDR_SCRIPT=$PWD/rd.bash
echo "source \"$CDR_SCRIPT\"" >> ~/.bashrc

# Start the rd daemon
killall -q rd
nohup rd -daemon 2>/dev/null &

echo "Setup complete. Start a new shell to use the 'cdr' command"
