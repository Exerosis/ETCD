#!/usr/bin/env bash
cd ../go-ycsb || exit
./bin/go-ycsb load etcd -p etcd.endpoints="192.168.1.1:2379,192.168.1.2.2379,192.168.1.3:2379,192.168.1.4:2379,192.168.1.5:2379" -P workloads/workloada_gryff
./bin/go-ycsb run etcd -p etcd.endpoints="192.168.1.1:2379,192.168.1.2.2379,192.168.1.3:2379,192.168.1.4:2379,192.168.1.5:2379" -P workloads/workloada_gryff