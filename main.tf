provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  region     = "${var.aws_region}"
}

resource "aws_s3_bucket" "cache" {
  bucket        = "${var.cache_repos["s3_bucket"]}"
  force_destroy = true

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET"]
    allowed_origins = ["*"]
  }
}

resource "aws_lambda_function" "cache_repos" {
  function_name = "cache-github-repositories"
  role          = "${aws_iam_role.lambda_assume.arn}"

  filename = "cache_repos/handler.zip"
  handler  = "handler.Handle"
  runtime  = "python2.7"

  source_code_hash = "${base64sha256(file("cache_repos/handler.zip"))}"

  environment = {
    variables = {
      GITHUB_ACCESS_TOKEN = "${var.github_access_token}"
      GITHUB_ORGANIZATION = "${var.cache_repos["github_organization"]}"
      S3_BUCKET           = "${var.cache_repos["s3_bucket"]}"
      S3_KEY              = "${var.cache_repos["s3_key"]}"
    }
  }
}

resource "aws_cloudwatch_event_rule" "trigger_cache_repos" {
  name        = "trigger-github-repositories-cache"
  description = "Write a JSON file of repos to S3 on a schedule"

  schedule_expression = "rate(30 minutes)"
}

resource "aws_cloudwatch_event_target" "trigger_cache_repos_target" {
  rule = "${aws_cloudwatch_event_rule.trigger_cache_repos.name}"
  arn  = "${aws_lambda_function.cache_repos.arn}"
}

resource "aws_iam_role" "lambda_assume" {
  name = "lambda-assume-role"

  assume_role_policy = "${data.aws_iam_policy_document.lambda_assume_role.json}"
}

resource "aws_iam_role_policy" "lambda_iam_policy" {
  name = "lambda-function-cache-github-repositories-access"

  role   = "${aws_iam_role.lambda_assume.id}"
  policy = "${data.aws_iam_policy_document.cache_repos_role_policy.json}"
}

resource "aws_lambda_permission" "allow_cloudwatch" {
  statement_id = "permit-cloudwatch-trigger-of-cache-github-repositories"

  action    = "lambda:InvokeFunction"
  principal = "events.amazonaws.com"

  source_arn    = "${aws_cloudwatch_event_rule.trigger_cache_repos.arn}"
  function_name = "${aws_lambda_function.cache_repos.function_name}"
}

data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "cache_repos_role_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "s3:PutObject",
      "s3:PutObjectAcl"
    ]
    resources = ["${aws_s3_bucket.cache.arn}/*"]
  }
}




