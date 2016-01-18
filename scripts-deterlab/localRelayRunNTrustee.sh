#!/bin/bash

source ~/config.sh

if [ $# -eq 0 ]
  then
    echo "First argument must be ntrustee id, numeric"
  exit 1
fi
cellsize=" "
if [ $# -eq 2 ]
  then
    echo "Second argument argument given, cellsize=$2"
  cellsize="-upcellsize=$2 "
fi

echo "Killing processess named ${program}..."
pkill -f ${program}

echo "Starting the relay with -ntrustees=$1, $cellsize $logParamsString log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -relay -ntrustees=$1 $tXhostsString $cellsize $logParamsString 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."