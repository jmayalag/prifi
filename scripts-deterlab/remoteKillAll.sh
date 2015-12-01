#max trustee minus one, really
maxtrustee=10
maxclient=10

echo "Remoting inside relay"
ssh relay.LB-LLD.SAFER.isi.deterlab.net "pkill -f prifi"

# Start clients and client
for i in $(seq 0 $maxclient); do
  echo "Remoting inside client-$i"
  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "pkill -f prifi"
done

# Start clients and trustees
for i in $(seq 0 $maxtrustee); do
  echo "Remoting inside trustee-$i"
  ssh trustee-$i.LB-LLD.SAFER.isi.deterlab.net "pkill -f prifi"
done