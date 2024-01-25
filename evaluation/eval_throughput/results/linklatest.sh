#!/usr/bin/env bash

for dir in $(find -type d -name wasimoff -o -name serverledge); do
  (cd "$dir"; ln -snf "$(ls -d1 1* | sort -n | tail -1)" latest;)
done
