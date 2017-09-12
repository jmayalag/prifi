#!/bin/sh
pid=$(ps -ejf | awk '/[\.]\/prifi.sh /{print $2}')
if [ ! -z "$pid" ]; then
    kill -- -$pid
    echo "Done, killed GID $pid."
else
    echo "Nothing to kill."
fi
