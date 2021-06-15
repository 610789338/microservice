#!/bin/bash

ps -ef|grep service_a|grep -v grep|awk '{print $2}'|xargs kill -9
nohup ./service_a -c config.json > service_a.log &
