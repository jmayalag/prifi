#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
ntrustee=3
total=61440

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [$repeat/$nrepeat]"

	for upCellSize in 1024 61440; do #$(seq 1024 1024 61440); do
		downCellSize=`expr $total - $upCellSize`

		echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 5

		echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Starting relay with $ntrustee trustees upCellSize $upCellSize downCellSize $downCellSize"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $upCellSize $downCellSize"
		echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 10 sec for relay to setup..."
		sleep 10
  
     	echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Starting client-0 upCellSize $upCellSize downCellSize $downCellSize"
		ssh client-0.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh 0 $upCellSize $downCellSize"
		echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 30 sec..."
		sleep 30

	done
done


echo "[$repeat/$nrepeat] Killing everything..."
/users/lbarman/dissent/remoteKillAll.sh
echo "All done."