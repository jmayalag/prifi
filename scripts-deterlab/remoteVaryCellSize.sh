#!/usr/local/bin/bash

#max trustee minus one, really
maxclient=9
nrepeat=9
ntrustee=3
maxcellsize=5120

for repeat in $(seq 0 $maxclient); do

	echo "Repetition [$repeat/$nrepeat]"

	for cellSize in $(seq 128 128 $maxtrustees); do


		echo "[$repeat/$nrepeat] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[$repeat/$nrepeat] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 10

		echo "[$repeat/$nrepeat] Starting relay with $ntrustee trustees cellsize $cellSize"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $cellSize"
		echo "[$repeat/$nrepeat] Waiting 60 sec for relay to setup..."
		sleep 60

		# Start clients
		for i in $(seq 0 $maxclient); do
		  echo "[$repeat/$nrepeat] Starting client-$i cellsize $cellSize"
		  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i $cellSize"
		  echo "[$repeat/$nrepeat] Waiting 30 sec before starting next client..."
		  sleep 30
		done
	done
done


echo "[$repeat/$nrepeat] Killing everything..."
/users/lbarman/dissent/remoteKillAll.sh
echo "All done."