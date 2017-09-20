#!/bin/bash
cd ..

rm -f rerun.log

./prifi.sh socks-d >> rerun.log 2>&1
./prifi.sh relay-d >> rerun.log 2>&1
./prifi.sh trustee-d 0 >> rerun.log 2>&1

cat rerun.log