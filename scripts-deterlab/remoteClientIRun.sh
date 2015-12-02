if [ $# -eq 0 ]
  then
    echo "First argument must be client id, numeric"
  exit 1
fi

echo "Remoting inside client-$1"
ssh client-$1.LB-LLD.SAFER.isi.deterlab.net "./dissent/localClientRun.sh $1"