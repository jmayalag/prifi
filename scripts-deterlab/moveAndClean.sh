rm *.log
rm *.out
rm dissent/prifi
rm dissent/*.sh
cp ./prifi/scripts-deterlab/* ./dissent/
cp ./prifi/bin/prifi ./dissent/
chmod u+rwx ./dissent/*