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

resource "aws_key_pair" "my_key" {
    key_name = "my_key"
    public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCvVLR1ocMxAOiwK5KYnAsZ1PZJtbX0mnRNRAh9D0kaDsj9pYJDvIlnVGCa7bE04xyTDCF5eAeNo8LDDaPXJ0RIP82owMhRSUw2D786AYB7/zDKwPjZvGG2WsE7cllnmRoJbqha8ZWMlr1KfqaFtyaRrDmqtk23R+vum/8biVBLlL8uoen9Mt+YAUyCASR8EdLkfAUP1Scjv0cIlQdHg67CUoEL2qJY5E8Vsji9w164o1CUsqDVEqkolHjR3SPizQuDuKDRWjVuFxjiVpa+IbMqDm7g0EviLo0EiMHVNAoyBqbKlbn8Pz9OR8qWpmUpVEtQR4263gtUm7kap9y+BPJZ"
}

resource "aws_security_group" "my_sg" {
    
    name = "my_sg"

    ingress {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}


resource "aws_security_group" "my_sg_http" {
    
    name = "my_sg_http"

    ingress {
        from_port = 8000
        to_port = 8000
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }

    egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}
resource "aws_instance" "my_server" {
    ami           = "ami-0c398cb65a93047f2"
    instance_type = "c7i-flex.large"

    key_name = aws_key_pair.my_key.key_name

    vpc_security_group_ids = [aws_security_group.my_sg.id, aws_security_group.my_sg_http.id]

    user_data = file("./startup.sh")

    tags = {
    Name = "Orchestrator"
    }
}

resource "aws_instance" "agent" {
    ami           = "ami-0c398cb65a93047f2"
    instance_type = "t3.small"

    key_name = aws_key_pair.my_key.key_name

    vpc_security_group_ids = [aws_security_group.my_sg.id, aws_security_group.my_sg_http.id]

    user_data = file("./startup.sh")

    tags = {
    Name = "Agent"
    }
}


# resource "aws_apigatewayv2_api" "api" {
#   name          = "ec2-http-api"
#   protocol_type = "HTTP"
# }

resource "aws_apigatewayv2_integration" "integration" {
  api_id                 = "kzvijk5asj"
  integration_type       = "HTTP_PROXY"
  integration_method     = "ANY"
  integration_uri        = "http://${aws_instance.my_server.public_ip}:8000"
  payload_format_version = "1.0"
}

# Catch-all route
resource "aws_apigatewayv2_route" "route" {
  api_id    = "kzvijk5asj"
  route_key = "$default"
  target    = "integrations/${aws_apigatewayv2_integration.integration.id}"
}

# Default stage (auto deploy enabled)
resource "aws_apigatewayv2_stage" "stage" {
  api_id      = "kzvijk5asj"
  name        = "$default"
  auto_deploy = true
}


output "orchestrator_public_ip" {
  value = aws_instance.my_server.public_ip
}

output "agent_public_ip" {
  value = aws_instance.agent.public_ip
}

output "aws_gateway_url" {
  value = "https://kzvijk5asj.execute-api.us-east-1.amazonaws.com/"
}