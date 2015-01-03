#!/bin/bash
# bootstrap chains - used to regenerate tests/chains/*.chain

mkdir -p chains
bash ./mine.sh 00 15
bash ./mine.sh 01 5 00
bash ./mine.sh 02 5 00
bash ./mine.sh 03 5 01
bash ./mine.sh 04 5 01