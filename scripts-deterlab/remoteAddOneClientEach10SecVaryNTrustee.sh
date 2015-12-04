#max trustee minus one, really
maxclient=9
nrepeat=9
maxtrustees=5


for repeat in $(seq 0 $maxclient); do

	echo "Repetition [$repeat/$nrepeat]"

	for ntrustee in $(seq 1 $maxtrustees); do

		echo "[$repeat/$nrepeat] Starting relay with $ntrustee trustees"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "./dissent/localRelayRunNTrustee.sh $ntrustee"
		echo "[$repeat/$nrepeat] Waiting 60 sec for relay to setup..."
		sleep 60

		# Start clients
		for i in $(seq 0 $maxclient); do
		  echo "[$repeat/$nrepeat] Starting client-$i"
		  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i"
		  echo "[$repeat/$nrepeat] Waiting 30 sec before starting next client..."
		  sleep 30
		done

		echo "[$repeat/$nrepeat] Killing all the clients..."

		for i in $(seq 0 $maxclient); do
		  echo "[$repeat/$nrepeat] Killing client-$i"
		  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "pkill -f prifi"
		done

		echo "[$repeat/$nrepeat] Killing relay"
		ssh router.LB-LLD.SAFER.isi.deterlab.net "pkill -f prifi"
		echo "[$repeat/$nrepeat] Waiting 10 sec before starting relay"
		sleep 10
	done
done