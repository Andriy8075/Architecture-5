#!/bin/bash

# Create database directory if it doesn't exist
mkdir -p /opt/practice-4/db

# Check if command exists
if [ ! -f "/opt/practice-4/$1" ]; then
    echo "Error: Command /opt/practice-4/$1 not found!"
    exit 127
fi

exec "/opt/practice-4/$1" "${@:2}"