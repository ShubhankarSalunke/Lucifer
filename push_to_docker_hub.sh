#!/bin/bash

# Push Docker images to Docker Hub
# This script assumes you have already built the Docker images using the deploy.sh script
echo "Orchestrator Docker Hub Push"
docker tag lucifer-orchestrator:latest shubhankarsal/lucifer-orchestrator:latest
docker push shubhankarsal/lucifer-orchestrator:latest

echo "Agent Docker Hub Push"
docker tag lucifer-agent:latest shubhankarsal/lucifer-agent:latest
docker push shubhankarsal/lucifer-agent:latest