#max trustee minus one, really
maxtrustee=4

# Start clients and trustees
for i in $(seq 0 $maxtrustee); do
  echo "Remoting inside trustee-$i"
  ssh trustee-$i.LB-LLD.SAFER.isi.deterlab.net "./dissent/localTrusteeSrvRun.sh $i"
done