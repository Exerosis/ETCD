#!/bin/bash
go run ./tools/benchmark \
--endpoints=http://192.168.1.2:2379 \
--conns=1 --clients=1 put --key-size=8 --total=500 --val-size=256
#--sequential-keys