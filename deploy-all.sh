#!/bin/sh

errorMsg="\e[31m\e[1m[error]\e[97m\e[0m"
okMsg="\e[32m[ok]\e[97m"

echo -n "Starting relay...	"
./run-prifi.sh relay > relay.log 2>&1 &
RELAYPID=$!
echo -e "$okMsg"

sleep 3

echo -n "Starting trustee 0...	"
./run-prifi.sh trustee 0 > trustee0.log 2>&1 &
TRUSTEE0PID=$!
echo -e "$okMsg"

sleep 3

echo -n "Starting client 0...	"
./run-prifi.sh client 0 > client0.log 2>&1 &
CLIENT0PID=$!
echo -e "$okMsg"

sleep 3

echo -n "Starting client 1...	"
./run-prifi.sh client 1 > client1.log 2>&1 &
CLIENT1PID=$!
echo -e "$okMsg"

sleep 3

read -p "PriFi deployed. Press [enter] to kill all..." key

kill $RELAYPID 2>/dev/null
kill $TRUSTEE0PID 2>/dev/null
kill $CLIENT0PID 2>/dev/null
kill $CLIENT1PID 2>/dev/null

echo -e "Script done."