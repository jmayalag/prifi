#!/bin/bash

source ~/config.sh

id=$1
if [ $# -eq 0 ]
  then
    echo "First argument not given, ID=0"
  id=0
fi
upcellsize=" "
if [ $# -eq 2 ]
  then
    echo "Second argument argument given, upcellsize=$2"
  upcellsize="-upcellsize=$2 "
fi
downcellsize=" "
if [ $# -eq 3 ]
  then
    echo "Second argument argument given, downcellsize=$3"
  downcellsize="-downcellsize=$3 "
fi

echo "Killing processess named ${program}..."
pkill -f ${program}
echo "Starting client ${id}, socks=$socks, relayhostaddr=$relayhostaddr $cellsize $logParamsString log redirected to ${nohupoutfolder}${nohupclientname}${id}${nohupext}..."
nohup "${programpath}${program}" -client=$1 -socks=$socks -relayhostaddr=$relayhostaddr $upcellsize $downcellsize $logParamsString 1>>${nohupoutfolder}${nohupclientname}${id}${nohupext} 2>&1 &
echo "Done."