#!/bin/bash
docker run \
       --detach \
       -p 9090:9090 \
       --name=go-cache-example \
       bose/go-cache-example

#       -e "REDIS_SENTINEL_ADDRESS=redis://host.docker.internal:26379" \
#       -e "REDIS_ADDRESS=redis://host.docker.internal:6379" \
