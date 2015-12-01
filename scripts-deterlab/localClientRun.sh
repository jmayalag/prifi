#!/bin/bash
echo "Killing processess..."
pkill -f prifi
echo "Removing old log files..."
rm -f client.stdout
rm -f cleit.stderr
echo "Starting client $1..."
nohup ~/dissent/prifi -client=$1 -socks=false -relayhostaddr=10.0.1.5:9876 1>>"client$1.out" 2>&1 &
echo "Done."