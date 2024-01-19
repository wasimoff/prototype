# This Dockerfile builds a custom Serverledge image to compare TSP throughput to Wasimoff.
FROM grussorusso/serverledge-base

# copy the built tsp binary and use it as custom cmd
COPY tsp.static /tsp
COPY wrapper.sh /wrapper.sh
ENV CUSTOM_CMD "/bin/ash /wrapper.sh"
