#!/usr/bin/python3

# usage: cat individualpcaps_.gnudata | ./stats.py

from pathlib import Path
import sys
import seaborn as sns
import matplotlib.pyplot as plt
import numpy as np
import fileinput

def reject_outliers(data):
    m = 2
    u = np.mean(data)
    s = np.std(data)
    filtered = [e for e in data if (u - 2 * s < e < u + 2 * s)]
    return filtered

def try_parse_int(s, base=10):
  try:
    return int(s, base)
  except ValueError:
    return None

fileData = []
for line in fileinput.input():
    needle = "PCAPLog-individuals "
    if needle in line:
        line = line[line.find(needle) + len(needle):].replace('( ', '').replace(' )', '').strip()
        parts = line.split(':')
        key = parts[0].strip()
        data = parts[1].strip().split(';')
        data = [try_parse_int(x) for x in data if x != ""]
        #fileData[key] = data
        fileData.extend(data)
        
plt.plot(reject_outliers(fileData))

plt.ylabel('Latency');
plt.xlabel('Packets');
plt.legend(loc='best')
t = 'mean ' + str(round(np.mean(fileData))), '; dev' + str(round(np.std(fileData)));
plt.title(t);
plt.show();