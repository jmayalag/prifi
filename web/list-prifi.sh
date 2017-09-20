#!/bin/sh
cd ..

echo "Reading rerun.log..."
echo ""
cat rerun.log
echo ""
echo "Done."

echo ""
echo ""

echo "Listing /prifi/ processes..."
echo ""
ps -u | grep /[p]rifi
echo ""
echo "Done."

echo ""
echo ""

echo "Listing log files..."
echo ""
ls -al --block-size=MB *.log
echo ""
echo "Done."