source ${programpath}config.sh

echo "Killing processess named ${program}..."
pkill -f ${program}
echo "Starting client $1, socks=$socks, relayhostaddr=$relayhostaddr, logtype=$netlogger -loghost=$loghost log redirected to ${nohupoutfolder}${nohupclientname}${nohupext}..."
nohup "${programpath}${program}" -client=$1 -socks=$socks -relayhostaddr=$relayhostaddr -logtype=$netlogger -loghost=$loghost 1>>${nohupoutfolder}${nohupclientname}${nohupext} 2>&1 &
echo "Done."