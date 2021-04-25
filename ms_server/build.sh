#!/bin/bash

if [ -e ~/go_devs/tmp/src ]; then
    rm ~/go_devs/tmp/src
fi

cd ..
ln -s $(pwd) ~/go_devs/tmp/src
cd -
go build
rm ~/go_devs/tmp/src
