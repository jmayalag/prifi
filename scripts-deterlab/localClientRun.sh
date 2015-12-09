echo "Killing processess..."
pkill -f prifi
echo "Starting client $1..."
nohup ~/dissent/prifi -client=$1 -socks=false -relayhostaddr=10.0.0.254:9876 -logtype=netlogger -loghost=192.168.253.1:10000 &
echo "Done."