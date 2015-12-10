source ~/config.sh

id=$1
if [ $# -eq 0 ]
  then
    echo "First argument not given, ID=0"
  id=0
fi

echo "Killing processess named ${program}..."
pkill -f ${program}


echo "Starting the trustee server $1, $logParamsString,  log redirected to ${nohupoutfolder}${nohuptrusteesrvname}${id}${nohupext}..."
nohup "${programpath}${program}" -trusteesrv $logParamsString 1>>${nohupoutfolder}${nohuptrusteesrvname}${id}${nohupext} 2>&1 &
echo "Done."