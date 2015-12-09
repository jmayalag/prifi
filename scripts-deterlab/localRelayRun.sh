echo "Killing processess..."
pkill -f prifi
echo "Starting the relay..."
nohup ~/dissent/prifi -relay -t1host=10.0.1.1:9000 -t2host=10.0.1.2:9000 -t3host=10.0.1.3:9000 -t4host=10.0.1.4:9000 -t5host=10.0.1.4:9000 -logtype=netlogger -loghost=192.168.253.1:10000 &
echo "Done."