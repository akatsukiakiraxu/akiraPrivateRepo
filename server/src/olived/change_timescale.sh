#!/bin/bash

DURATION=$1
HORIZONTAL_POINTS=$2
if [ -z "$DURATION" ]; then DURATION=1.0; fi
if [ -z "$HORIZONTAL_POINTS" ]; then HORIZONTAL_POINTS=1000; fi
curl -H "Content-Type: application/json" -d "{\"duration\":$DURATION, \"horizontal_points\":$HORIZONTAL_POINTS}" http://localhost:2223/monitoring/settings/set
