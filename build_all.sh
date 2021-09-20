#!/bin/bash

echo "build service_a..."
cd service_a;go build -p 8
cd ..

echo "build service_b..."
cd service_b;go build -p 8
cd ..

echo "build ms_gate..."
cd ms_gate/src;./build.sh
cd ../..

echo "build push_service..."
cd push_service/src;./build.sh
cd ../..

echo "build client..."
cd client;go build -p 8
