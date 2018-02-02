#!/usr/bin/env python3

import sys
from tempfile import mkstemp
from shutil import move
from os import fdopen, remove

def replace(file_path, toChange):
    #Create temp file
    fh, abs_path = mkstemp()
    with fdopen(fh,'w') as new_file:
        with open(file_path) as old_file:
            for line in old_file:
                parts = line.split('=');
                if len(parts) == 2:
                    key = parts[0].strip()
                    val = parts[1].strip()
                    replaced = False
                    for key2 in toChange:
                        if key==key2:
                            line = key + ' = '  + toChange[key2]+'\n';
                            replaced = True
                    if not replaced:
                        line = key + ' = '  + val + '\n';
                new_file.write(line)
    #Remove original file
    remove(file_path)
    #Move new file
    move(abs_path, file_path)

toChange = {};
for p in sys.argv[1:]:
    if not '=' in p:
        print("Please provide the parameters to change, as key=val");
        sys.exit(1);
    parts = p.split('=');
    toChange[parts[0]] = parts[1];

if len(toChange) == 0:
    print("Please provide the parameters to change, as key=val");
    sys.exit(1);

replace("prifi_simul.toml", toChange)
