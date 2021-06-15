#!/bin/bash


ps -ef|grep ms_gate|grep -v grep|awk '{print $2}'|xargs kill -9

nohup ./ms_gate -c config.json > ms_gate.log &
