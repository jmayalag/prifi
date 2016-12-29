#!/usr/bin/env bash

#variables
colors="true"
dbg_lvl=3

# first argument can be used to replace the port
port="8090"
if [ "$#" -eq 1 ]; then
	port="$1"
fi

echo -e "Running a socks server on port $port, debug level $dbg_lvl/5."

DEBUG_COLOR=$colors go run run-server.go -debug="$dbg_lvl" -port="$port"
