terraform {
  backend "s3" {
    bucket = "cacophony-terraform"
    key    = "deployment-gateway"
    region = "us-east-1"
  }
}

# Wew, environment variable configuration
provider "aws" {
  version = "~> 1.18"
  region  = "us-east-1"
}

