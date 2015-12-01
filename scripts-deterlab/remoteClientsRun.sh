#max trustee minus one, really
maxtrustee=1

# Start clients and trustees
for i in $(seq 0 $maxtrustee); do
  echo "Remoting inside client-$i"
  ssh client-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $i"
done