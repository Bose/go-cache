#!/bin/bash

redis-cli -p 6379 SHUTDOWN; redis-cli -p 6380 SHUTDOWN; redis-cli -p 6381 SHUTDOWN; redis-cli -p 26379 SHUTDOWN; redis-cli -p 5000 SHUTDOWN; redis-cli -p 5001 SHUTDOWN;

#redis-cli -p 6379 SHUTDOWN; redis-cli -p 6380 SHUTDOWN; redis-cli -p 6381 SHUTDOWN; redis-cli -p 26379 SHUTDOWN
# redis-cli -p 5001 SHUTDOWN; redis-cli -p 5002 SHUTDOWN
