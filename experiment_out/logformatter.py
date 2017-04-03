#!/usr/bin/python2

import sys
import os
import json
import operator
import re

regex1str = "\[([\d]+)\] ([\d\.]+) round\/sec, ([\d\.]+) kB\/s up, ([\d\.]+) kB\/s down, ([\d\.]+) kB\/s down\(udp\)"
regex1 = re.compile(regex1str)
regex2str = "\[([\d]+)\] ([\d\.]+) ms \+- ([\d\.]+) \(over ([\d\.]+), happened ([\d\.]+)\)\. Info: ([\w\-_]+)"
regex2 = re.compile(regex2str)

data = []

def openFile(directory, fileName):
	with open(directory + fileName) as file:
		rawData = file.read()
		lines = rawData.split("\n")

		#find the last report, i.e. the steady state
		maxReportId = -1
		for line in lines:
			reportIdMatches = re.findall("\[([\d]+)\]", line)
			if len(reportIdMatches) == 0:
				continue
			reportId = reportIdMatches[0]
			if int(reportId) > maxReportId:
				maxReportId = int(reportId)

		print "Max report id is "+str(maxReportId)

		# filter by latest report (most stable)
		interestingData = []
		for line in lines:
			if "["+str(maxReportId)+"]" in line:
				interestingData.append(line)

		# parse the data
		for line in interestingData:
			parsed = regex1.findall(line)
			if len(parsed) > 0:
				parsed = parsed[0]
				throughputData = {}
				throughputData["fileName"] = fileName
				throughputData["directory"] = directory
				throughputData["reportId"] = parsed[0]
				throughputData["round_s"] = parsed[1]
				throughputData["tp_up"] = parsed[2]
				throughputData["tp_down"] = parsed[3]
				throughputData["tp_udp_down"] = parsed[4]
				data.append(throughputData)
			parsed = regex2.findall(line)
			if len(parsed) > 0:
				parsed = parsed[0]
				durationData = {}
				durationData["fileName"] = fileName
				durationData["directory"] = directory
				durationData["reportId"] = parsed[0]
				durationData["duration_mean"] = parsed[1]
				durationData["duration_conf"] = parsed[2]
				durationData["nsamples"] = parsed[3]
				durationData["popsize"] = parsed[4]
				durationData["text"] = parsed[5]
				data.append(durationData)

#main

if len(sys.argv) < 2:
	print "Argument 2 must be the folder with the logs"
	sys.exit(1)


logFolder = str(sys.argv[1])
if not logFolder.endswith("/"):
	logFolder = logFolder + "/"
print "Processing log folder "+logFolder


# list all files in dir
files = []
for (dirpath, dirnames, filenames) in os.walk(logFolder):
	files = filenames
	break

for file in files:
	if file == "config" or file == "compiled.json":
		continue
	print "Processing "+file
	openFile(logFolder, file)
		
# store it in a JSON file
print json.dumps(data, sort_keys=False)

with open(logFolder + "compiled.json", "w") as myfile:
	myfile.write(json.dumps(data, sort_keys=False))