if [ $# -eq 0 ]
  then
    echo "First argument must be trustee id, numeric"
  exit 1
fi
echo "Killing processess..."
pkill -f prifi
echo "Starting the trustee server $1..."
nohup ~/dissent/prifi -trusteesrv -logtype=netlogger -loghost=192.168.253.1:10000 &
echo "Done."