#!/usr/bin/python3

# usage: tcpdump -ttttnnr file.pcap | ./stats.py

def try_parse_int(s, base=10):
  try:
    return int(s, base)
  except ValueError:
    return None

import fileinput
from pprint import pprint
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import sys

nPackets = 0;
sizes = [];
histogram = {}
for line in fileinput.input():
    
    needle = "length"
    if needle in line:
        pos = line.find(needle)
        length = line[pos + len(needle) + 1:].strip()
        parsedLength = try_parse_int(length)

        if parsedLength != None:
            nPackets += 1;
            sizes.append(parsedLength)
            if not parsedLength in histogram:
                histogram[parsedLength] = 0;
            histogram[parsedLength] += 1

plt.hist(sizes, normed=False, bins=30)
plt.xlabel('Packet Size');
plt.xlabel('Number of packets');
plt.title("Total packets: " + str(nPackets));
plt.show();
