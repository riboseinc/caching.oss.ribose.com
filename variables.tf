variable "aws_access_key" {}
variable "aws_secret_key" {}
variable "aws_region" {
  default = "us-east-1"
}

variable "github_access_token" {}

variable "cache_repos" {
  type = "map"
  default = {
    github_organization = "riboseinc"
    s3_bucket           = "caching.oss.ribose.com"
    s3_key              = "repos.json"
  }
}
