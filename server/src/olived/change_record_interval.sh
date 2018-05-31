#!/bin/bash

STORE_ALWAYS=false
INTERVAL=$1
if [ -z "$INTERVAL" ]; then INTERVAL=0; STORE_ALWAYS=true; fi
curl -H "Content-Type: application/json" -d "{\"store_always\":$STORE_ALWAYS,\"store_every_seconds\":$INTERVAL,\"channels\":{\"ch1\":{\"enabled\":true},\"ch2\":{\"enabled\":true},\"ch3\":{\"enabled\":true},\"ch4\":{\"enabled\":true}}}" http://localhost:2223/recording/settings/set
