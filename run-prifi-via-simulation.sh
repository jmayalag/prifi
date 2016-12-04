#!/bin/bash
errorMsg="\e[31m\e[1m[error]\e[97m\e[0m"

echo -e "$errorMsg not supported."
exit

lvl=3
if [ $# -eq 1 ]
  then
    lvl=$1
fi
echo "Make sure $GOPATH/src/github.com/dedis/cothority is a git repo, on branch prifi"
echo "Make sure $GOPATH/src/github.com/lbarman/prifi_dev is the git repo of prifi"
echo "Make sure $GOPATH/src/github.com/dedis/ [cothority|crypto] are pulled to the latest version"
echo "---"
currentPath=$(pwd)
echo "Running PriFi simulation through SDA, debug level is $lvl, output is in $currentPath/log.txt"
cd $GOPATH/src/github.com/dedis/cothority/simul;
go build
./simul -debug $lvl runfiles/prifi_simple.toml -platform localhost 2>&1 | tee "$currentPath"/log.txt
