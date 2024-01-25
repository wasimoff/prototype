#!/usr/bin/env python3
# transform four individual files to a nice tabular presentation for latex

import pandas as pd

# read input files for scenarios
ts = "1696194946"
scenarios = [ "native", "static_musl", "wasmtime", "wasimoff" ]
tables = { sc: pd.read_csv(f"results/{ts}/tspfine_{ts}_{sc}.csv", index_col="parameter_n") for sc in scenarios }

# format a tabular text that can be pasted
print(f"$n$ & {' & '.join(tables.keys())} \\\\")
for n in [6, 7, 8, 9, 10, 11, 12]:
  print(f"{n}", end="")
  for (sc, table) in tables.items():
    result = table.loc[n]
    mean = result["mean"] * 1000
    stdd = result["stddev"] * 1000
    print(f" & ${mean:.2f} \\pm {stdd:.2f}$", end="")
  print(" \\\\")