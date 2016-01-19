#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
maxclient=5
ntrustee=3
total=61440

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [$repeat/$nrepeat]"

	for upCellSize in $(seq 10240 10240 61440); do
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

		# Start clients
		for i in $(seq 0 $maxclient); do
		  echo "[$repeat/$nrepeat][$upCellSize|$downCellSize]  Starting client-$i upCellSize $upCellSize downCellSize $downCellSize"
		  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i $upCellSize $downCellSize"
		  echo "[$repeat/$nrepeat][$upCellSize|$downCellSize]  Waiting 20 sec before starting next client..."
		  sleep 20
		done

		echo "[$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 30 sec..."
		sleep 30

	done
done


echo "[$repeat/$nrepeat] Killing everything..."
/users/lbarman/dissent/remoteKillAll.sh
echo "All done."