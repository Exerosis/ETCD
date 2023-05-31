#!/bin/bash
cd ../Raft || exit
git stash && git stash clear && git pull
cd ../RabiaGo || exit
git stash && git stash clear && git pull
cd ../PineappleGo || exit
git stash && git stash clear && git pull
cd ../ETCD || exit
git stash && git stash clear && git pull

HOST=$(hostname | awk -F "." '{print $1}')
echo "Hostname: $HOST"

if [ $HOST = "node-1" ]; then
    IP="192.168.1.1"
elif [ $HOST = "node-2" ]; then
    IP="192.168.1.2"
elif [ $HOST = "node-3" ]; then
    IP="192.168.1.3"
fi
echo "Local IP: $IP"

sudo rm -rf "$HOST.etcd"
make build
sudo ./bin/etcd --log-level panic \
--name "$HOST" \
--initial-cluster-token etcd-cluster-1 \
--listen-client-urls http://"$IP":2379,http://127.0.0.1:2379 \
--advertise-client-urls http://"$IP":2379 \
--initial-advertise-peer-urls http://"$IP":12380 \
--listen-peer-urls http://"$IP":12380 \
--initial-cluster node-1=http://192.168.1.1:12380,node-2=http://192.168.1.2:12380,node-3=http://192.168.1.3:12380 \
--initial-cluster-state new



