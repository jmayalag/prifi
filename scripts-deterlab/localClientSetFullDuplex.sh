
#!/bin/bash

source ~/config.sh

int=$(ip route show | grep 'default via 10.0.0.254' | cut -d ' '  -f 5)
echo "Interface to router is $int"
echo "Setting half duplex..."
echo "Exec sudo ethtool -s $int duplex full autoneg off"
sudo ethtool -s $int duplex full autoneg off