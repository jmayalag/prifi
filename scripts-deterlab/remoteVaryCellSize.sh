#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
ntrustee=3
maxcellsize=5120

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [$repeat/$nrepeat]"

	for cellSize in $(seq 128 128 $maxcellsize); do

		echo "[$repeat/$nrepeat] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[$repeat/$nrepeat] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 10

		echo "[$repeat/$nrepeat] Starting relay with $ntrustee trustees cellsize $cellSize"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $cellSize"
		echo "[$repeat/$nrepeat] Waiting 60 sec for relay to setup..."
		sleep 20
  
     	echo "[$repeat/$nrepeat] Starting client-0 cellsize $cellSize"
		ssh client-0.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh 0 $cellSize"
		echo "[$repeat/$nrepeat] Waiting 30 sec..."
		sleep 30

	done
done


echo "[$repeat/$nrepeat] Killing everything..."
/users/lbarman/dissent/remoteKillAll.sh
echo "All done."