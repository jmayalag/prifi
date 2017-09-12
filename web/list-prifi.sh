#!/bin/sh
cd ..

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
