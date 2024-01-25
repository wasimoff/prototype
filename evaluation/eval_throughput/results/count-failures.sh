#!/usr/bin/env bash
# extract the total number of failures per project across all runs

for project in wasimoff serverledge; do
  echo -n "$project: "
  for dir in $(find -type d -name $project); do (
    cd "$dir/latest";
    if [[ $(wc -l < locust_failures.csv) -gt 1 ]]; then
      tail -1 locust_failures.csv | sed 's/.*,//';
    fi
  ); done \
  | awk 'BEGIN { sum = 0 } { sum += $1 } END { print sum }';
done
