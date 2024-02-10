#!/bin/sh
set -exu

docker-compose -f docker-compose-add-member.yml --profile settlement down
