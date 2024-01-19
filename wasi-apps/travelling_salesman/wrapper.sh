#!/usr/bin/env ash
# wrapper script for custom serverledge image to read params and write result
# https://github.com/grussorusso/serverledge/blob/main/docs/custom_runtime.md

# the sed invocations to parse parameters here will add *some* overhead .. but the
# overall execution time should still be dominated by the tsp problem

# read the parameter from file
#! this expects only {"n": "12"}
n=$(sed -n 's/^{"n": \?"\?\([0-9]\+\)"\?}$/\1/p' < "$PARAMS_FILE")

# run the travelling salesman
/tsp rand "$n" > "$RESULT_FILE"
