source ~/config.sh

#max trustee minus one, really
maxtrustee=9
maxclient=4

echo "Remoting inside relay"
ssh router.LB-LLD.SAFER.isi.deterlab.net "pkill -f ${program}"

# Start clients and client
for i in $(seq 0 $maxclient); do
  echo "Remoting inside client-$i"
  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "pkill -f ${program}"
done

# Start clients and trustees
for i in $(seq 0 $maxtrustee); do
  echo "Remoting inside trustee-$i"
  ssh trustee-$i.LB-LLD.SAFER.isi.deterlab.net "pkill -f ${program}"
done