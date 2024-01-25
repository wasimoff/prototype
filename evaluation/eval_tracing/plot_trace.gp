#!/usr/bin/gnuplot

# input file is CSV
INPUT = "averagedStepsLoaded_n64_tsp8"
set datafile separator ";" missing NaN

# output: PDF
pdf_size = 3.5; pdf_ratio = 1.6;
set terminal pdfcairo enhanced color lw 1 solid size pdf_size*pdf_ratio,pdf_size font "Helvetica,11"
ext = ".pdf"

# output: SVG
#set terminal svg enhanced size 800,500 dynamic background rgb "white" font ",15"
#ext = ".svg"

# output: PNG
#set terminal pngcairo size 800, 500
#ext = ".png"

# axes and labels ...
# set xlabel "Step"
set ylabel "Average time in milliseconds"
set key top left
set key at screen 0.17,0.92
set logscale y
set grid

# set style data histograms
set style histogram errorbars gap 2
set style fill solid 0.8 border -1
set boxwidth 0.85
set xtics rotate by 90 right
# set yrange [0:1.7]

# plot using clustered histogram bars with errorlines
set output "trace_".INPUT.ext
mycolor(x) = ((x*11244898) + 2851770)
fmtf(x) = sprintf("%.2f", x)
plot \
  INPUT.".csv" using 0:1:(mycolor($3)):xtic(2) with boxes title "" lc variable, \
  '' every ::1 using ($0+1):(6):(fmtf($1)) with labels title "", \
  # '' every ::8::8 using (8):(1.53):1 with labels title "", \
  # '' every ::13::13 using (13):(1.53):1 with labels title "", \