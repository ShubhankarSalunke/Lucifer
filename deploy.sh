#!/bin/bash

# Lucifer AWS Deployment Script

set -e

echo "Starting Lucifer deployment to AWS..."

# 1. Build Docker images
echo "Building Docker images..."

# Build from root directory with full context including dependencies
# Build orchestrator image
docker build -t lucifer-orchestrator:latest -f chaos-engineering/orchestrator/Dockerfile .

# Build agent image
docker build -t lucifer-agent:latest -f chaos-engineering/agent/Dockerfile .

echo "Docker images built successfully."

# # 2. Initialize Terraform
# echo "Initializing Terraform..."
# cd infrastructure
# terraform init

# # 3. Apply Terraform infrastructure
# echo "Applying Terraform configuration..."
# terraform apply -auto-approve

# # Get outputs
# ORCHESTRATOR_REPO=$(terraform output -raw orchestrator_repository_url)
# AGENT_REPO=$(terraform output -raw agent_repository_url)
# ORCHESTRATOR_IP=$(terraform output -raw ec2_public_ip)
# API_ID=$(terraform output -raw api_id)

# echo "Orchestrator Repository: $ORCHESTRATOR_REPO"
# echo "Agent Repository: $AGENT_REPO"
# echo "Orchestrator IP: $ORCHESTRATOR_IP"
# echo "API Gateway ID: $API_ID"

# # 4. Authenticate Docker to ECR
# echo "Authenticating Docker to ECR..."
# aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $ORCHESTRATOR_REPO

# # 5. Tag and push images
# echo "Tagging and pushing images..."

# # Tag orchestrator
# docker tag lucifer-orchestrator:latest $ORCHESTRATOR_REPO:latest
# docker push $ORCHESTRATOR_REPO:latest

# # Tag agent
# docker tag lucifer-agent:latest $AGENT_REPO:latest
# docker push $AGENT_REPO:latest

# echo "Images pushed to ECR."

# # 6. Deploy containers to EC2 instances
# echo "Deploying containers to EC2..."

# # For orchestrator instance
# ORCHESTRATOR_INSTANCE_ID=$(aws ec2 describe-instances --filters "Name=tag:Name,Values=Orchestrator" "Name=instance-state-name,Values=running" --query "Reservations[0].Instances[0].InstanceId" --output text)

# # For agent instance
# AGENT_INSTANCE_ID=$(aws ec2 describe-instances --filters "Name=tag:Name,Values=Agent" "Name=instance-state-name,Values=running" --query "Reservations[0].Instances[0].InstanceId" --output text)

# # Run orchestrator on orchestrator instance
# aws ssm send-command --document-name "AWS-RunShellScript" --instance-ids $ORCHESTRATOR_INSTANCE_ID --parameters commands="
# docker pull $ORCHESTRATOR_REPO:latest
# docker run -d --name lucifer-orchestrator -p 8000:8000 $ORCHESTRATOR_REPO:latest
# " --output text

# # Run agent on agent instance
# aws ssm send-command --document-name "AWS-RunShellScript" --instance-ids $AGENT_INSTANCE_ID --parameters commands="
# docker pull $AGENT_REPO:latest
# docker run -d --name lucifer-agent -e CONTROL_PLANE=http://$ORCHESTRATOR_IP:8000 $AGENT_REPO:latest
# " --output text

# echo "Deployment completed!"
# cd ..

# echo "Orchestrator API available at: http://$ORCHESTRATOR_IP:8000"
# echo "API Gateway endpoint: https://$API_ID.execute-api.us-east-1.amazonaws.com/"