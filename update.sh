#!/usr/bin/env bash
cd ../Raft || exit
git pull
cd ../RabiaGo || exit
git pull
cd ../ETCD || exit
git pull

make build
sudo ./bin/etcd --log-level panic \
--name infra0 \
--initial-cluster-token etcd-cluster-1 \
--initial-advertise-peer-urls http://192.168.1.1:12380 \
--listen-client-urls http://192.168.1.1:2379 \
--advertise-client-urls http://192.168.1.1:2379 \
--listen-peer-urls http://192.168.1.1:12380 \
--initial-cluster infra0=http://192.168.1.1:2380,infra1=http://192.168.1.2:2380,infra2=http://192.168.1.3:2380 \
--initial-cluster-state new



