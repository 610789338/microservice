#!/bin/bash


cd

rm -Rf ~/go
wget https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz
tar -zxvf go1.13.1.linux-amd64.tar.gz

rm -Rf ~/go_devs
mkdir -p ~/go_devs/framework/src
mkdir -p ~/go_devs/tmp

apt-get install git
go get go.etcd.io/etcd/client/v3
go get github.com/vmihailenco/msgpack

echo "export PATH=$(pwd)/go/bin:"'$PATH' >> ~/.profile
echo "export GOPATH=$(pwd)/go_devs/framework:$(pwd)/go_devs/tmp" >> ~/.profile
source ~/.profile

