# /gateway/main.tf
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  required_version = ">= 1.5.0"
}

provider "aws" {
  region = "us-east-1"
}

variable "region" {
  default = "us-east-1"
}

resource "aws_apigatewayv2_api" "api" {
  name          = "ec2-http-api"
  protocol_type = "HTTP"
}

# We output the ID so the EC2 folder can "find" it later
output "api_id" {
  value = aws_apigatewayv2_api.api.id
}