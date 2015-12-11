#!/usr/local/bin/bash

#max trustee minus one, really
maxclient=9
nrepeat=9


for repeat in $(seq 0 $maxclient); do

	echo "Repetition [$repeat/$nrepeat]"

	# Start clients
	for i in $(seq 0 $maxclient); do
	  echo "[$repeat/$nrepeat] Starting client-$i"
	  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i"
	  echo "[$repeat/$nrepeat] Waiting 10 sec before starting next client..."
	  sleep 10
	done

	echo "[$repeat/$nrepeat] Killing all the clients..."

	for i in $(seq 0 $maxclient); do
	  echo "[$repeat/$nrepeat] Killing client-$i"
	  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "pkill -f prifi"
	done

	echo "[$repeat/$nrepeat] Waiting 30 sec for relay to resync..."
	sleep 30
done