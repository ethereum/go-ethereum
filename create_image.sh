#!/bin/bash

# Set the name of the container you want to create an image from
container_name="limechain-task-geth-1"

# Get the container ID by filtering the output of `docker ps`
container_id=$(docker ps -qf "name=$container_name")

# Check if the container ID is not empty
if [ -n "$container_id" ]; then
  echo "Found container ID: $container_id"

  # Generate a unique tag for the new image
  image_tag="anglyuboslav/geth-smart-con"

  # Create an image from the container
  docker commit "$container_id" "$image_tag"

  echo "Image created with tag: $image_tag"
else
  echo "No running container found with name: $container_name"
fi