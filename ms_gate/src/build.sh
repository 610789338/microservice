#!/bin/bash

app="ms_gate"

go build -o ${app}
mv ${app} ../bin/

