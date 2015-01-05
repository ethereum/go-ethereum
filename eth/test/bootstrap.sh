#!/bin/bash
# bootstrap chains - used to regenerate tests/chains/*.chain

mkdir -p chains
bash ./mine.sh 00 10
bash ./mine.sh 01 5 00
bash ./mine.sh 02 10 00
bash ./mine.sh 03 5 02
bash ./mine.sh 04 10 02