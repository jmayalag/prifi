source ~/config.sh

echo "Killing processess named ${program}..."
pkill -f ${program}
echo "Starting the relay, logtype=$netlogger -loghost=$loghost log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -relay -t1host=$t1host -t2host=$t2host -t3host=$t3host -t4host=$t4host -t5host=$t5host -logtype=$netlogger -loghost=$loghost 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."