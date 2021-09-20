#!/bin/bash

app="ms_gate"

go build -o ${app} -p 8
mv ${app} ../bin/

