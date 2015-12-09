echo "Killing processess..."
pkill -f prifi
echo "Deleting old log files..."
rm -f /tmp/sink.log
echo "Starting log sink..."
nohup ~/dissent/prifi-freebsd-amd64 -logsink -socks=false -logpath=/tmp/ &
echo "Done."