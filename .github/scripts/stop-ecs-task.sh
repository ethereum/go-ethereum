#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Invalid number of arguments. Usage: stop-ecs-task.sh <cluster> <service>"
    exit 1
fi

cluster=$1
service=$2

tasks=$(aws ecs list-tasks --cluster $cluster --service-name $service)

task_arn=$(echo $tasks | awk -F\[ '{print $2}' | awk -F\" '{print $2}')

if [ -n "${task_arn}" ]; then
  aws ecs stop-task --cluster $cluster --task $task_arn > /dev/null
fi