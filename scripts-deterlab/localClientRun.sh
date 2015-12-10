source config.sh

echo "Killing processess..."
pkill -f ${program}
echo "Starting client $1..."
nohup ${programpath}${program} -client=$1 -socks=$socks -relayhostaddr=$relayhostaddr -logtype=$netlogger -loghost=$loghost &
echo "Done."