source ~/config.sh

id=$1
if [ $# -eq 0 ]
  then
    echo "First argument not given, ID=0"
  id=0
fi

echo "Killing processess named ${program}..."
pkill -f ${program}
echo "Starting client ${id}, cmd=nohup "${programpath}${program}" socks=$socks, relayhostaddr=$relayhostaddr, logtype=$netlogger -loghost=$loghost log redirected to ${nohupoutfolder}${nohupclientname}${id}${nohupext}..."
nohup "${programpath}${program}" -client=$1 -socks=$socks -relayhostaddr=$relayhostaddr -logtype=$netlogger -loghost=$loghost 1>>${nohupoutfolder}${nohupclientname}${id}${nohupext} 2>&1 &
echo "Done."