#!/bin/bash

set -e

IMAGE_NAME=adahealth/k8s-oidc-helper

echo "building ${IMAGE_NAME}..."
sudo docker build -t $IMAGE_NAME .
sudo docker login -u 'adabot' -p $DOCKER_PASSWORD
sudo docker push $IMAGE_NAME
