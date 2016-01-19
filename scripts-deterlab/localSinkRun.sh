#!/usr/local/bin/bash

source ~/config.sh

echo "Killing processess named ${programFreeBSD}..."
pkill -f ${programFreeBSD}

echo "Deleting old log file ${logPath}${logsinkname}"
rm -f "${logPath}${logsinkname}"
rm -f "${nohupoutfolder}${nohupsinkname}${nohupext}"

echo "Starting log sink -logpath=$logPath, log redirected to ${nohupoutfolder}${nohupsinkname}${nohupext}..."
nohup "${programpath}${programFreeBSD}" -logsink -logpath=$logPath 1>>${nohupoutfolder}${nohupsinkname}${nohupext} 2>&1 &
echo "Done."

#tail -f ${nohupoutfolder}${nohupsinkname}${nohupext}