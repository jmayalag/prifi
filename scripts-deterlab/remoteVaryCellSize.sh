#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
ntrustee=3
maxcellsize=61440
downCellSize=40960
window=1

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [$repeat/$nrepeat]"

	for upCellSize in $(seq 2048 2048 $maxcellsize); do

		echo "[$repeat/$nrepeat][$downCellSize/$maxcellsize] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[$repeat/$nrepeat][$downCellSize/$maxcellsize] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 5

		echo "[$repeat/$nrepeat][$downCellSize/$maxcellsize] Starting relay with $ntrustee trustees, upCellSize $upCellSize downCellSize $downCellSize window $window"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee  $upCellSize $downCellSize $window"
		echo "[$repeat/$nrepeat][$downCellSize/$maxcellsize] Waiting 10 sec for relay to setup..."
		sleep 10
  
     	echo "[$repeat/$nrepeat][$downCellSize/$maxcellsize] Starting client-0  upCellSize $upCellSize downCellSize $downCellSize"
		ssh client-0.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh 0 $upCellSize $downCellSize"
		echo "[$repeat/$nrepeat][$downCellSize/$maxcellsize] Waiting 30 sec..."
		sleep 30

	done
done


echo "[$repeat/$nrepeat] Killing everything..."
/users/lbarman/dissent/remoteKillAll.sh
echo "All done."