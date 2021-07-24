#!/bin/bash

cd

rm -Rf ~/go
wget https://dl.google.com/go/go1.14.1.linux-amd64.tar.gz
tar -zxvf go1.14.1.linux-amd64.tar.gz

# use go mod
echo "export PATH=$(pwd)/go/bin:"'$PATH' >> ~/.profile
echo "export GO111MODULE=on" >> ~/.profile
source ~/.profile

