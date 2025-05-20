#!/bin/bash

# Додамо перевірку на наявність команди
if [ ! -f "/opt/practice-4/$1" ]; then
    echo "Error: Command /opt/practice-4/$1 not found!"
    exit 127
fi

exec "/opt/practice-4/$1" "${@:2}"