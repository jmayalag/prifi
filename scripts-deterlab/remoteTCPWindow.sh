#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
ntrustee=3
maxwindow=10

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [TCP-$repeat/$nrepeat]"

	for window in $(seq 1 1 $maxwindow); do
		for downCellSize in $(seq 10240 10240 61440); do
			upCellSize=1024

			echo "[TCP-$repeat/$nrepeat][$window][$downCellSize] Killing everything..."
			/users/lbarman/dissent/remoteKillAll.sh

			echo "[TCP-$repeat/$nrepeat][$window][$downCellSize] Starting the trustees..."
			/users/lbarman/dissent/remoteTrusteesSrvRun.sh
			sleep 5

			echo "[TCP-$repeat/$nrepeat][$window][$downCellSize] Starting relay with $ntrustee trustees upCellSize $upCellSize downCellSize $downCellSize window $window"
			ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $upCellSize $downCellSize $window"
			echo "[TCP-$repeat/$nrepeat][$window][$downCellSize] Waiting 10 sec for relay to setup..."
			sleep 5

			echo "[TCP-$repeat/$nrepeat][$window][$downCellSize]  Starting client-0 upCellSize $upCellSize downCellSize $downCellSize"
			ssh client-0.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh 0 $upCellSize $downCellSize"

			echo "[TCP-$repeat/$nrepeat][$window][$downCellSize] Waiting 20 sec..."
			sleep 20

		done
	done
done

cp /tmp/sink.nohup /tmp/sink_TCP_WINDOW.nohup