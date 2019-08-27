#!/bin/bash

mkdir -p ./run
cp ./conf/* ./run
redis-cli -p 6380 SLAVEOF localhost 6379; redis-cli -p 6381 SLAVEOF localhost 6379; cd ./run && (redis-sentinel sentinel1.conf & redis-sentinel sentinel2.conf & redis-sentinel sentinel3.conf)
