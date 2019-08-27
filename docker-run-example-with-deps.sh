#!/bin/bash
./sentinel-run.sh > sentinel.out &
./redis-run.sh > redis.out &

# wait for redis and sentinel to be ready...
sleep 20

./example
