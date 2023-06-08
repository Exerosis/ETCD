#!/bin/bash
HOST_1="http://192.168.1.1:2379"
HOST_2="http://192.168.1.2:2379"
HOST_3="http://192.168.1.3:2379"
go run ./tools/benchmark \
--endpoints=${HOST_1},${HOST_2},${HOST_3} --conns=100 --clients=1000 \
    range YOUR_KEY --consistency=s --total=100000