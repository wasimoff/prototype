#!/usr/bin/env python
# pull stats from multiple runs into a single csv for plotting

# useful series:
# - many-tiny:       ./pull_histogram_stats.py -l nbg1-fsn1-fsn1 -c 04x02c -c 08x02c -c 12x02c -c 16x02c -c 20x02c -c 24x02c -c 28x02c -c 32x02c > results/histogram_lots-of-tiny.csv
# - medium regions:  ./pull_histogram_stats.py -l nbg1-fsn1-fsn1 -l hil-fsn1-fsn1 -l nbg1-fsn1-ash -c 02x08c -c 04x08c -c 06x08c -c 08x08c > results/histogram_some-medium-regions.csv
# - biglittle:       ./pull_histogram_stats.py -l nbg1-fsn1-fsn1 -c 48c+16c -c 02x32c -c 04x16c -c 08x08c -c 16x04c -c 32x02c > results/histogram_biglittle.csv
# - tspn:            ./pull_histogram_stats.py -l nbg1-fsn1-fsn1 -c 32x02c -n 8 -n 9 -n 10 -n 11 > results/histogram_tspn.csv  (and then edit 2nd column to tsp-n labels)

import csv, sys, argparse

parser = argparse.ArgumentParser()
parser.add_argument("-l", dest="locations", help="location strings", action="append")
parser.add_argument("-c", dest="corespecs", help="core configs", action="append")
parser.add_argument("-n", dest="tspn", help="tsp n parameters", action="append")
parser.add_argument("-p", dest="projects", help="projects to compare", action="append")
args = parser.parse_args()

# defaults to compare different sized ccx13 clusters
if args.locations is None: args.locations = [ "nbg1-fsn1-fsn1" ]
if args.corespecs is None: args.corespecs = [ "04x02c", "08x02c", "12x02c", "16x02c", "20x02c", "24x02c", "28x02c", "32x02c" ]
if args.tspn      is None: args.tspn      = [ "8", "10" ]
if args.projects  is None: args.projects  = [ "wasimoff", "serverledge" ]

# known interesting fieldnames in locust stats
fName = "Name"
fRequests = "Request Count"
fFailures = "Failure Count"
fRttMedian = "Median Response Time"
fRttAvg = "Average Response Time"
fRttMin = "Min Response Time"
fRttMax = "Max Response Time"
fReqsPs = "Requests/s"
fFailPs = "Failures/s"
fQ25 = "25%"
fQ50 = "50%"
fQ75 = "75%"
fQ80 = "80%"
fQ90 = "90%"
fQ95 = "95%"
fQ98 = "98%"
fQ99 = "99%"
fQ100 = "100%"

# open a csv writer on stdout
fieldnames = [ "Scenario", fName, fRequests, fReqsPs, fFailures, fFailPs, fRttMin, fRttAvg, fRttMedian, fRttMax, fQ99 ]
out = csv.DictWriter(sys.stdout, delimiter=",", fieldnames=fieldnames)
out.writeheader()

# lambda to assemble the correct file path to locust stats
stats = lambda l, c, n, p: f"results/{l}/{c}/tsp-{n}/{p}/latest/locust_stats.csv"

# lambda to print back the identifier for a row
rowid = lambda l, c, n, p: f"{l}/{c}/tsp-{n}/{p}"
name = lambda l, c, n, p: f"{c}"

# nested iteration over all the requested runs
for l in args.locations:
  for c in args.corespecs:
    for n in args.tspn:
      for p in args.projects:

        file = csv.DictReader(open(stats(l, c, n, p), "rt"))
        for row in file:
          if row[fName] == "Aggregated":
            sub = { col: row.get(col) for col in fieldnames }
            sub["Scenario"] = rowid(l, c, n, p)
            sub[fName] = name(l, c, n, p)
            out.writerow(sub)
