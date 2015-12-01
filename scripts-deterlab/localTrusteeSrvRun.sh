#!/bin/bash
if [ $# -eq 0 ]
  then
    echo "First argument must be trustee id, numeric"
  exit 1
fi
echo "Killing processess..."
pkill -f prifi
echo "Removing old log files..."
rm -f "trustee$1.out"
echo "Starting the trustee server $1..."
nohup ~/dissent/prifi -trusteesrv 1>>"trustee$1.out" 2>&1 &
echo "Done."