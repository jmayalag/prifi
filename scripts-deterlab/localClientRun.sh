#!/bin/bash

source ~/config.sh

id=$1
if [ $# -eq 0 ]
  then
    echo "First argument not given, ID=0"
  id=0
fi
upcellsize=" "
if (( $# > 1 ))
  then
    echo "Second argument given, upcellsize=$2"
  upcellsize="-upcellsize=$2 "
fi
downcellsize=" "
if (( $# > 2 ))
  then
    echo "Third argument given, downcellsize=$3"
  downcellsize="-downcellsize=$3 "
fi

echo "Killing processes named ${program}..."
pkill -f ${program}
echo "Starting client ${id}, socks=$socks, relayhostaddr=$relayhostaddr $cellsize $logParamsString log redirected to ${nohupoutfolder}${nohupclientname}${id}${nohupext}..."
nohup "${programpath}${program}" -node=prifi-client-$1 -socks=$socks -relayhostaddr=$relayhostaddr $upcellsize
$downcellsize $logParamsString 1>>${nohupoutfolder}${nohupclientname}${id}${nohupext} 2>&1 &
echo "Done."