#!/usr/local/bin/bash

#max trustee minus one, really
nrepeat=9
maxclient=5
ntrustee=3
total=61440

sed -i -- 's/useUdp="false"/useUdp="true"/g' config.sh

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [UDP-$repeat/$nrepeat]"

	for upCellSize in $(seq 10240 10240 61440); do
		downCellSize=`expr $total - $upCellSize`

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 5

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Starting relay with $ntrustee trustees upCellSize $upCellSize downCellSize $downCellSize"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $upCellSize $downCellSize"
		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 10 sec for relay to setup..."
		sleep 10

		# Start clients
		for i in $(seq 0 $maxclient); do
		  echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize]  Starting client-$i upCellSize $upCellSize downCellSize $downCellSize"
		  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i $upCellSize $downCellSize"
		  echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize]  Waiting 20 sec before starting next client..."
		  sleep 20
		done

		echo "[UDP-$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 30 sec..."
		sleep 30

	done

	cp /tmp/sink.nohup /tmp/sink_UDP_${repeat}.nohup
done

cp /tmp/sink.nohup /tmp/sink_UDP_FINAL.nohup

echo "Switching to TCP"
/users/lbarman/dissent/remoteKillAll.sh
/users/lbarman/dissent/localSinkRun.sh
sleep 60

sed -i -- 's/useUdp="true"/useUdp="false"/g' config.sh

for repeat in $(seq 0 $nrepeat); do

	echo "Repetition [TCP-$repeat/$nrepeat]"

	for upCellSize in $(seq 10240 10240 61440); do
		downCellSize=`expr $total - $upCellSize`

		echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize] Killing everything..."
		/users/lbarman/dissent/remoteKillAll.sh

		echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize] Starting the trustees..."
		/users/lbarman/dissent/remoteTrusteesSrvRun.sh
		sleep 5

		echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize] Starting relay with $ntrustee trustees upCellSize $upCellSize downCellSize $downCellSize"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee $upCellSize $downCellSize"
		echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 10 sec for relay to setup..."
		sleep 10

		# Start clients
		for i in $(seq 0 $maxclient); do
		  echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize]  Starting client-$i upCellSize $upCellSize downCellSize $downCellSize"
		  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i $upCellSize $downCellSize"
		  echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize]  Waiting 20 sec before starting next client..."
		  sleep 20
		done

		echo "[TCP-$repeat/$nrepeat][$upCellSize|$downCellSize] Waiting 30 sec..."
		sleep 30

	done

	cp /tmp/sink.nohup /tmp/sink_TCP_${repeat}.nohup
done
cp /tmp/sink.nohup /tmp/sink_TCP_FINAL.nohup


echo "[$repeat/$nrepeat] Killing everything..."
/users/lbarman/dissent/remoteKillAll.sh
echo "All done."