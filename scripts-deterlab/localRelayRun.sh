source ~/config.sh

echo "Killing processess named ${program}..."
pkill -f ${program}
echo "Starting the relay, $logParamsString log redirected to ${nohupoutfolder}${nohuprelayname}${nohupext}..."
nohup "${programpath}${program}" -relay $tXhostsString $logParamsString 1>>${nohupoutfolder}${nohuprelayname}${nohupext} 2>&1 &
echo "Done."