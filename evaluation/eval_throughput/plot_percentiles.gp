#!/usr/bin/gnuplot

# PLOT ALL:
# for loc in results/*-*-*; do
#   for cores in "$loc"/*c; do
#     l=$(basename "$loc");
#     c=$(basename "$cores");
#     for tsp in tsp-8 tsp-10; do
#       gnuplot -e "LOCATION = '$l'" -e "CORES = '$c'" -e "TSPN = '$tsp'" plot_percentiles.gp;
#     done;
#   done;
# done;

# https://github.com/Gnuplotting/gnuplot-palettes/tree/master
# colors from spectral palette
c0 = '#3288BD' # blue
c1 = '#66C2A5' # green
c2 = '#ABDDA4' # pale green
c3 = '#E6F598' # pale yellow-green
c4 = '#FEE08B' # pale yellow-orange
c5 = '#FDAE61' # pale orange
c6 = '#F46D43' # orange
c7 = '#D53E4F' # red
# c7 = '#f097a1' # lighter red

# output: EPS
eps_size = 3.0
eps_ratio = 1.6
# set terminal postscript eps enhanced dashed lw 0.5 color size eps_size*eps_ratio,eps_size font "Helvetica,14"
# ext = ".eps"

# output: PDF
set terminal pdfcairo enhanced color lw 1 solid size eps_size*eps_ratio,eps_size font "Helvetica,11"
ext = ".pdf"

# output: SVG
# set terminal svg enhanced size 800,500 dynamic background rgb "white" font ",15"
# ext = ".svg"

# output: PNG
# set terminal pngcairo size 800, 500
# ext = ".png"

#! when changing scenario, check interesting yrange too!
if (!exists("LOCATION"))  LOCATION = "nbg1-fsn1-fsn1";
if (!exists("CORES"))     CORES = "32x02c";
if (!exists("TSPN"))      TSPN = "tsp-10";
SCENARIO = LOCATION."/".CORES."/".TSPN
set output "plots/percentiles_".LOCATION."_".CORES."_".TSPN.ext
set label 1 at screen 0.32, 0.97 SCENARIO

# load files for wasimoff and serverledge
set datafile separator ","
csv = "locust_stats_history.csv"
fWasi = "results/".SCENARIO."/wasimoff/latest/".csv
fServ = "results/".SCENARIO."/serverledge/latest/".csv

# compute stats to calculate run time from start
stats fWasi using 1:5 name "sWasi"
stats fServ using 1:5 name "sServ"

# interesting ranges depend on scenario
if (TSPN eq "tsp-8") || (TSPN eq "tsp-9") set yrange [10:30000]; \
else if (TSPN eq "tsp-10")                set yrange [100:80000]; \
else                                      set yrange [1000:80000];
set xrange [:159]

# axes and labels
set xlabel "Time (s)"
set ylabel "Response Time (ms)"
set key right top outside
set grid ytics mytics y x ls -1
set logscale y # don't see much otherwise
set style fill solid 0.8 border -1

# set seconds tics with reduced frequency
set xtics 0, 30, 160

set style fill transparent solid 0.5

# OLD: plot reponse time percentiles over run time as lines
# plot FILE using ($1-data_min_x):7 title columnheader(7), \
#   for [i=8:15] '' using i title columnheader(i)

# arrange wasimoff and serverledge side-by-side
set multiplot layout 1, 2
set tmargin at screen 0.86
set lmargin at screen 0.14
set rmargin at screen 0.44
set nokey

set title "Wasimoff"
plot fWasi using ($1-sWasi_min_x):20:7  with lines lc rgb c0 lw 2 title "Minimum", \
        "" using ($1-sWasi_min_x):7:9   with filledcurves lc rgb c1 title sprintf("%s-%s", columnhead(7), columnhead(9)), \
        "" using ($1-sWasi_min_x):9:10  with filledcurves lc rgb c2 title columnhead(10), \
        "" using ($1-sWasi_min_x):10:11 with filledcurves lc rgb c3 title columnhead(11), \
        "" using ($1-sWasi_min_x):11:12 with filledcurves lc rgb c4 title columnhead(12), \
        "" using ($1-sWasi_min_x):12:13 with filledcurves lc rgb c5 title columnhead(13), \
        "" using ($1-sWasi_min_x):13:14 with filledcurves lc rgb c6 title columnhead(14), \
        "" using ($1-sWasi_min_x):21 with lines lc rgb c7 lw 2 title "Maximum", \
        "" using ($1-sWasi_min_x):19 with lines lc -1 lw 2 title "Average"

set key at screen 0.98,0.87
set lmargin at screen 0.46
set rmargin at screen 0.76
set ylabel ""
set format y ""

fWasi = fServ
sWasi_min_x = sServ_min_x
set title "Serverledge"
replot
# plot fServ using ($1-sServ_min_x):20:7  with lines lc rgb c0 lw 2 title "Minimum", \
#         "" using ($1-sServ_min_x):7:9   with filledcurves lc rgb c1 title sprintf("%s-%s", columnhead(7), columnhead(9)), \
#         "" using ($1-sServ_min_x):9:10  with filledcurves lc rgb c2 title columnhead(10), \
#         "" using ($1-sServ_min_x):10:11 with filledcurves lc rgb c3 title columnhead(11), \
#         "" using ($1-sServ_min_x):11:12 with filledcurves lc rgb c4 title columnhead(12), \
#         "" using ($1-sServ_min_x):12:13 with filledcurves lc rgb c5 title columnhead(13), \
#         "" using ($1-sServ_min_x):13:14 with filledcurves lc rgb c6 title columnhead(14), \
#         "" using ($1-sServ_min_x):21 with lines lc rgb c7 lw 2 title "Maximum", \
#         "" using ($1-sServ_min_x):19 with lines lc -1 lw 2 title "Average"
        