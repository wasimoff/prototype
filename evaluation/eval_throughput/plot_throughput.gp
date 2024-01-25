#!/usr/bin/gnuplot

# output: PDF
pdf_size = 3.0; pdf_ratio = 1.6;
set terminal pdfcairo enhanced color lw 1 solid size pdf_size*pdf_ratio,pdf_size font "Helvetica,11"
ext = ".pdf"

# output: SVG
# set terminal svg enhanced size 800,500 dynamic background rgb "white" font ",15"
# ext = ".svg"

# output: PNG
# set terminal pngcairo size 800, 500
# ext = ".png"


# specify data to plot
# for project { for corespec { for tspn { plot line } } }
LOCATION  = "nbg1-fsn1-fsn1" # currently, use only one location
PROJECTS  = "wasimoff serverledge"

# compare all 2-core scenarios
# CORESPECS = "04x02c 08x02c 12x02c 16x02c 20x02c 24x02c 28x02c 32x02c"
# TSPN      = "10"
# TITLE     = "Influence of the amount of CCX13 servers, TSP-10"

CORESPECS = "04x02c 08x02c 12x02c 16x02c"
TSPN      = "8"
TITLE     = ""
OUTPUT    = "plots/throughput_".LOCATION."_04,08,12,16x02c_tsp-".TSPN.ext
set label 1 at screen 0.78, 0.64 "All with tsp-8."

# compare all 8-core scenarios
# CORESPECS = "02x08c 04x08c 06x08c 08x08c"
# TSPN      = "10"
# TITLE     = "Influence of the amount of CCX33 servers"

# compare different biglittle slicings
# CORESPECS = "48c+16c 02x32c 08x08c 32x02c"
# TSPN      = "10"
# TITLE     = "Influence of different 64 vCPU core slicings"

# compare different TSP n parameters on single scenario
# CORESPECS = "32x02c"
# TSPN      = "8 9 10"
# TITLE     = "Influence of TSP parameter, 32x02c"

# assemble output name, if not specified
lastcore = word(CORESPECS, words(CORESPECS))
lasttspn = word(TSPN, words(TSPN))
if (!exists("OUTPUT")) OUTPUT = "plots/throughput_".LOCATION."_".lastcore."_tsp-".lasttspn.ext
set output OUTPUT
print "Plot to", OUTPUT

# set line colors to access by index
# https://iamkate.com/data/12-bit-rainbow/
colours = "#801070 #e09040 #90d050 #00b0c0 #3060b0 #c06060 #20c0b0 #603090"

# axes and labels
set datafile separator ","
set style data lines
set xlabel "Time (s)"
set ylabel "Throughput (req/s)"
set key right bottom outside
set key top right outside
set title TITLE

# collect vectors
files = "" # filenames to plot from
statv = "" # stats vector for min_x values
names = "" # names for line title

# collect line styles
dasht = "" # dash types
colrs = "" # line colors

# for project { for corespec { for tspn { plot line } } }
do for [p=1:words(PROJECTS)] {
  project = word(PROJECTS, p)
  do for [c=1:words(CORESPECS)] {
    spec = word(CORESPECS, c)
    do for [n=1:words(TSPN)] {
      tsp = word(TSPN, n)

      # dash type depends on project
      dasht = dasht." ".(((p-1)*3)+1)

      # line color depends on corespec x tsp
      colrs = colrs." ".((c-1)*words(TSPN)+n)

      # append data file to files vector
      histfile = "results/".LOCATION."/".spec."/tsp-".tsp."/".project."/latest/locust_stats_history.csv"
      files = files." ".histfile

      # append title to names vector
      name = "'".project.", ".spec."'" # for varying corespec
      #name = "'".project.", tsp-".tsp."'" # for varying tsp n
      names = names." ".name

      # append prefix of stats variables for this file to stats vector
      sv = "p".p."_c".c."_t".n."_stats"
      statv = statv." ".sv
      stats histfile using 1:5 name sv

    }
  }
}

# plot all the collected files
# using req/s over run time
plot for [i=1:words(files)] word(files, i) \
  using ($1-value(word(statv, i)."_min_x")):($5-$6) \
  title word(names, i) \
  with lines lw 2 lc rgb word(colours, 0+word(colrs, i)) dt 0+word(dasht, i) #ls 1 lw 3

