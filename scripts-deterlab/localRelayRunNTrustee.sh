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
window=" "
if (( $# > 3 ))
  then
    echo "Fourth argument argument given, window=$4"
  window="-window=$4 "
fi

echo "Killing processess named ${program}..."
pkill -f ${program}

echo "Starting the relay with -ntrustees=$1 $upcellsize $window -relaydummydown $downcellsize $logParamsString log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -relay -ntrustees=$1 $tXhostsString -relaydummydown $window $upcellsize $downcellsize $logParamsString 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."