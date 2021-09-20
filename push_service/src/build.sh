#!/bin/bash

app="push_service"

go build -o ${app} -p 8
mv ${app} ../bin/

