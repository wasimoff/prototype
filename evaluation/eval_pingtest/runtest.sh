#!/usr/bin/env bash
# measure latency between hetzner regions

# created one host in every region, ipv6 only
fsn="2a01:4f8:c012:34a4::1"
nbg="2a01:4f8:1c0c:7b82::1"
hel="2a01:4f9:c012:9ed7::1"
ash="2a01:4ff:f0:8758::1"
hil="2a01:4ff:1f0:8095::1"

# an array with all pairs
pairs=(fsn/nbg fsn/hel fsn/ash fsn/hil nbg/hel nbg/ash nbg/hil hel/ash hel/hil ash/hil)

# run pairwise pings and log them to txt files
mkdir -p results
for pair in "${pairs[@]}"; do

  # parse pair to locations
  from=${pair%/*};
  to=${pair#*/};
  out="results/$from-$to.txt";

  # log command to output
  printf '$ ssh %s ping -i 0.1 -c 256 %s\n' "$from" "$to" \
  | tee "$out"

  # run command and append log to output
  ssh "root@${!from}" ping -i 0.1 -c 256 "${!to}" \
  | tee -a "$out"

done

# three more as a reference from home
for dest in fsn hel ash; do
  out="results/home-$dest.txt";
  printf '$ ping -i 0.1 -c 256 %s\n' "$dest" | tee "$out";
  ping -i 0.1 -c 256 "${!dest}" | tee -a "$out";
done

# get the results to a csv file
pairs+=(home/fsn home/hel home/ash);
echo -n > results.csv
for pair in "${pairs[@]}"; do

  # parse pair to locations
  from=${pair%/*};
  to=${pair#*/};
  out="results/$from-$to.txt";

  # parse the rtt statistics: min/avg/max/mdev
  rtt=($(sed -nE 's|^rtt .+ = ([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+) ms.*|\1 \2 \3 \4|p' < "$out"));

  # append to csv
  printf "%s;%s;%s;%s;%s;%s\n" "$from" "$to" "${rtt[@]}" >> results.csv

done

# sort the file by avg and add header
sort --numeric --field-separator ";" --key 4 < results.csv \
| sed '1i src;dest;min;avg;max;mdev' \
| sponge results.csv