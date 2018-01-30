#!/usr/bin/python3

# usage: tcpdump -ttttnnr file.pcap | ./stats.py

def try_parse_int(s):
  try:
    return int(s, 10)
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

t0 = None

toSend = [];

totalPacketInput = 0
totalPacketSent = 0
PRIFI_BUCKET_SIZE = 2000 #B
PRIFI_PACKETS_EVERY_X_MS = 1
lastPriFiPacket = 0

for line in fileinput.input():
    
    totalPacketInput += 1

    parts = line.split(' ')
    timeStr = parts[1]
    timeParts = timeStr.split('.')
    timeParts2 = timeParts[0].split(':')
    m = int(timeParts2[1])
    s = int(timeParts2[2])
    ms = int(timeParts[1])
    ts = int(float(str(m * 60 * s)+"."+str(ms)) * 1000) # ms

    if try_parse_int(str(ts)) == None:
        continue

    if t0 == None:
        t0 = ts

    length = 1
    needle = "length"
    if needle in line:
        pos = line.find(needle)
        length2 = line[pos + len(needle) + 1:].strip()
        parsedLength = try_parse_int(length2)

        if parsedLength != None:
            length = parsedLength

    if length == 0:
        length = 1

    while length > PRIFI_BUCKET_SIZE:
        toSend.append(PRIFI_BUCKET_SIZE)
        length -= PRIFI_BUCKET_SIZE
    toSend.append(length)

    while len(toSend) > 0 and lastPriFiPacket + PRIFI_PACKETS_EVERY_X_MS < ts:
        currentBucket = 0;
        while len(toSend) > 0 and currentBucket + toSend[0] <= PRIFI_BUCKET_SIZE:
            s = toSend.pop()
            currentBucket += s
        lastPriFiPacket += PRIFI_PACKETS_EVERY_X_MS
        totalPacketSent+=1

print(totalPacketInput, totalPacketSent, int(totalPacketSent/float(totalPacketInput) * 100), "%")