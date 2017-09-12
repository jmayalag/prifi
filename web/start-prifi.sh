#!/bin/sh

cd ..
rm -f trustee0.log relay.log

nohup ./prifi.sh trustee 0 1>>trustee0.log 2>&1 &
nohup ./prifi.sh relay 1>>relay.log 2>&1 &
echo "Done, started Relay and Trustee 0"
