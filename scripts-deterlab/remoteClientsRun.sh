#!/usr/local/bin/bash

#max trustee minus one, really
maxclient=0

# Start clients and trustees
for i in $(seq 0 $maxclient); do
  echo "Remoting inside client-$i"
  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i"
done