#!/bin/bash
if [ $# -eq 0 ]
  then
    echo "First argument must be ntrustee id, numeric"
  exit 1
fi
echo "Killing processess..."
pkill -f prifi
echo "Removing old log files..."
rm -f relay.out
echo "Starting the relay with $1 trustees..."
nohup ~/dissent/prifi -relay -ntrustees=$1 -t1host=10.0.1.1:9000 -t2host=10.0.1.2:9000 -t3host=10.0.1.3:9000 -t4host=10.0.1.4:9000 -t5host=10.0.1.4:9000 1>>"relay.out" 2>&1 &
echo "Done."