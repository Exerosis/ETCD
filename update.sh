#!/usr/bin/env bash
git pull
make build
sudo ./bin/etcd --log-level panic \
--name=infra0 \
--initial-cluster infra0=http://192.168.1.1:2380,infra1=http://192.168.1.2:2380,infra2=http://192.168.1.3:2380 \
--initial-cluster-state new



