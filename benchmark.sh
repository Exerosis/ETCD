#!/bin/bash
go run ./tools/benchmark \
--endpoints=http://192.168.1.1:2379,http://192.168.1.2:2379,http://192.168.1.3:2379 \
--conns=500 --clients=500 put --key-size=8 --total=100000 --val-size=256
#--sequential-keyssadf