#!/usr/local/bin/bash

source ~/config.sh

#max trustee minus one, really
maxclient=9

# Configure clients
for i in $(seq 0 $maxclient); do
  echo "Remoting inside client-$i"
  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientSetHalfDuplex.sh"
done