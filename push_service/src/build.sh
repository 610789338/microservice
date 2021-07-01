#!/bin/bash

app="push_service"

go build -o ${app}
mv ${app} ../bin/

