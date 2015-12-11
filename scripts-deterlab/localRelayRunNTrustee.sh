#!/bin/bash

source ~/config.sh

if [ $# -eq 0 ]
  then
    echo "First argument must be ntrustee id, numeric"
  exit 1
fi

echo "Killing processess named ${program}..."
pkill -f ${program}

echo "Starting the relay with -ntrustees=$1, $logParamsString log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -relay -ntrustees=$1 $tXhostsString $logParamsString 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."