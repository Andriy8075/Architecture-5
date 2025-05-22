#!/bin/bash

# For db service only
if [ "$1" = "db" ]; then
    mkdir -p /data
fi

# Check if command exists
if [ ! -f "/opt/practice-4/$1" ]; then
    echo "Error: Command /opt/practice-4/$1 not found!"
    exit 127
fi

exec "/opt/practice-4/$1" "${@:2}"