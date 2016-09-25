#!/bin/bash

if [ -z "$GOPATH" ]; then
	echo "Error: GOPATH is unset."
	exit
fi
mkdir -p "$GOPATH/src/github.com/dedis"
cd "$GOPATH/src/github.com/dedis"

if [ ! -d "cothority" ]; then
	echo "Cloning github.com/dedis/cothority in your GOPATH"
	echo "..."
	git clone https://github.com/dedis/cothority.git
fi
cd "cothority"
git checkout prifi
git branch --set-upstream-to=origin/prifi prifi
git pull

branch=$(git rev-parse --abbrev-ref HEAD)

echo "..."
if [ $branch = "prifi" ]; then
	echo "Successfully checkout out branch PriFi, pulling..."
	git pull
else
	echo "Could not checkout branch PriFi, please do it manually"
fi

echo "Gonna download the dependancies..."
cd "simul"
go get

prifidev="$GOPATH/src/lbarman/prifi_dev/"
if [ -d "$prifidev" ]; then
	echo "$prifidev seems correctly setup"
else
	echo "Careful, you did not create $prifidev. You will not be able to run"
fi

currPath=$(pwd)
if [ "$currPath" = "$prifidev" ]; then
	echo "Seems that you are working in the prifi_dev folder under $GOPATH, it's OK!"
else
	echo "Careful, seems this repo ($currPath) is *NOT* in your GOPATH. It need to be the case, or you will not be able to edit the files and run ! if you're using a symlink, that's fine, otherwise move this repo (this folder) to $GOPATH/src/lbarman/prifi_dev."
fi
