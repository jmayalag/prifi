#!/bin/bash

source ~/config.sh

if [ $# -eq 0 ]
  then
    echo "First argument must be ntrustee id, numeric"
  exit 1
fi
upcellsize=" "
if (( $# > 1 ))
  then
    echo "Second argument argument given, upcellsize=$2"
  upcellsize="-upcellsize=$2 "
fi
downcellsize=" "
if (( $# > 2 ))
  then
    echo "Third argument argument given, downcellsize=$3"
  downcellsize="-downcellsize=$3 "
fi

echo "Killing processess named ${program}..."
pkill -f ${program}

echo "Starting the relay with -ntrustees=$1, $cellsize $logParamsString log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -relay -ntrustees=$1 $tXhostsString $upcellsize $downcellsize $logParamsString 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."