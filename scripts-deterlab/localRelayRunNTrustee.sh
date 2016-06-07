#!/bin/bash

source ~/config.sh

upcellsize=" "
if (( $# > 1 ))
  then
    echo "First argument argument given, upcellsize=$1"
  upcellsize="-upcellsize=$1 "
fi
downcellsize=" "
if (( $# > 2 ))
  then
    echo "Second argument argument given, downcellsize=$2"
  downcellsize="-downcellsize=$2 "
fi
window=" "
if (( $# > 3 ))
  then
    echo "Third argument argument given, window=$3"
  window="-window=$3 "
fi

echo "Killing processess named ${program}..."
pkill -f ${program}

echo "Starting the relay with -ntrustees=$1 $upcellsize $window -relaydummydown $downcellsize $logParamsString log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -node=prifi-relay $tXhostsString -relaydummydown $window $upcellsize
$downcellsize $logParamsString 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."