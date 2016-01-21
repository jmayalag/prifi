#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
ntrustee=3

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [UDP-$repeat/$nrepeat]"

	for downCellSize in $(seq 10240 10240 61440); do
		upCellSize=1024

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 5

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Starting relay with $ntrustee trustees upCellSize $upCellSize downCellSize $downCellSize"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $upCellSize $downCellSize"
		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 10 sec for relay to setup..."
		sleep 10

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize]  Starting client-0 upCellSize $upCellSize downCellSize $downCellSize"
		ssh client-0.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh 0 $upCellSize $downCellSize"

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 30 sec..."
		sleep 30

	done

	cp /tmp/sink.nohup /tmp/sink_UDP_${repeat}.nohup
done

cp /tmp/sink.nohup /tmp/sink_UDP_FINAL.nohup