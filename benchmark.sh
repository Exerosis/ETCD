#!/bin/bash
go run ./tools/benchmark \
--endpoints=http://192.168.1.1:2379,http://192.168.1.2:2379,http://192.168.1.3:2379 \
--conns=100 --clients=1000 put --key-size=8 --sequential-keys --total=100000 --val-size=256