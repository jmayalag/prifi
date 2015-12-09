rm *.log
rm *.out
rm dissent/prifi
rm dissent/*.sh
cp ./prifi/scripts-deterlab/* ./dissent/
cp ./prifi/bin/prifi-linux-amd64/prifi ./dissent/prifi
cp ./prifi/bin/prifi-freebsd-amd64/prifi ./dissent/prifi-freebsd-amd64
chmod u+rwx ./dissent/*