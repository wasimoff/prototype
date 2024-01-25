#!/usr/bin/gnuplot

# input file is CSV
set datafile separator "," missing NaN

# output: PDF
pdf_size = 3.0; pdf_ratio = 1.6;
set terminal pdfcairo enhanced color lw 1 solid size pdf_size*pdf_ratio,pdf_size font "Helvetica,11"
ext = ".pdf"

# output: SVG
#set terminal svg enhanced size 800,500 dynamic background rgb "white" font ",15"
#ext = ".svg"

# output: PNG
#set terminal pngcairo size 800, 500
#ext = ".png"

# axes and labels ...
set xlabel "Set size parameter n"
set ylabel "Runtime in seconds"
set key top left
set key at screen 0.17,0.92
set logscale y
set grid


# OLD PLOT COMMAND with lines
# columns: 1:command, 2:mean, 3:stddev, 4:median, 5:user, 6:system, 7:min, 8:max, 9:parameter_n
# set xrange [1 to 13.9] # do not show the 14 marker
# if (!exists("ts")) ts=1695802227
# set output "tspfine_".ts.ext
# plot \
#   "results/".ts."/tspfine_".ts."_native.csv"      using ($9-0.10):2:7:8 with errorlines lt 1 lw 1 pt 7 ps 0.6 title 'native',    \
#   "results/".ts."/tspfine_".ts."_static_musl.csv" using ($9-0.05):2:7:8 with errorlines lt 2 lw 1 pt 5 ps 0.6 title 'static',    \
#   "results/".ts."/tspfine_".ts."_wasmtime.csv"    using ($9-0.00):2:7:8 with errorlines lt 3 lw 1 pt 6 ps 0.6 title 'wasmtime',  \
#   "results/".ts."/tspfine_".ts."_wasimoff.csv"    using ($9+0.05):2:7:8 with errorlines lt 4 lw 1 pt 4 ps 0.6 title 'wasimoff'


set style data histograms
set style histogram errorbars gap 2
set style fill solid 0.8 border -1
set boxwidth 1.0
set xrange [-1:12]
set yrange [0.0001:1000]

# give timestamps to plot all in a loop
# 1695727781: first run, local
# 1695730754: longer run, local, tsp-12
# 1695802227: dedicated hetzner server, tsp-13
# 1695819786: looks botched, background during an event, local
# 1696194946: final attempt on my laptop, tsp-13
tss = "1695727781 1695730754 1695802227 1695819786 1696194946"
do for [ts in tss] {

  # plot using clustered histogram bars with errorlines
  set output "tspfine_".ts.ext
  plot \
    "results/".ts."/tspfine_".ts."_native.csv"      every 1::1 using 7:3:xtic(9) lt 1 lw 1 title 'native',    \
    "results/".ts."/tspfine_".ts."_static_musl.csv" every 1::1 using 7:3         lt 2 lw 1 title 'static',    \
    "results/".ts."/tspfine_".ts."_wasmtime.csv"    every 1::1 using 7:3         lt 3 lw 1 title 'wasmtime',  \
    "results/".ts."/tspfine_".ts."_wasimoff.csv"    every 1::1 using 7:3         lt 4 lw 1 title 'wasimoff'

}
