#!/bin/sh

# Script variables

debug_lvl=0
conf_file="config.toml"
group_file="group.toml"
bin_file="$GOPATH/src/github.com/lbarman/prifi_dev/sda/app/prifi.go"

print_usage() {
	echo "Usage: run.sh <role> <id>"
	echo "	Role: client, relay or trustee"
	echo "	Id: integer, only for relay or trustee roles"
}

test_digit() {
	case $1 in
		''|*[!0-9]*) print_usage; exit ;;
		*) ;;
	esac
}

# Argument validation

if [ "$#" -eq 1 ] && [ ! "$1" = "relay" ]; then
	print_usage
	exit
elif [ "$#" -eq 2 ]; then
	case "$1" in
		client|trustee) test_digit "$2" ;;
		*) print_usage; exit ;;
	esac
elif [ "$#" -eq 0 ] || [ "$#" -gt 2 ]; then
	print_usage
	exit
fi

# Check that config files exist

confdir="$PWD/$1$2"

if [ ! -f "$confdir/$conf_file" ]; then
	echo "Config file does not exist: $confdir/$conf_file"
	exit
fi

if [ ! -f "$confdir/$group_file" ]; then
	echo "Group file does not exist: $confdir/$group_file"
	exit
fi

# Run PriFi !

go run $bin_file -c "$confdir/$conf_file" -g "$confdir/$group_file" -d $debug_lvl "$1"

