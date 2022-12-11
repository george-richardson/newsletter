#!/bin/zsh
set -e

ARTIFACT_PATH="out/"
ZIP_PATH="out/newsletter.zip"
STACK_NAME="newsletter-dev"
HOSTED_ZONE_NAME="gjhr.me"
DOMAIN_NAME="newsletter-dev.gjhr.me"
CERTIFICATE_ARN="arn:aws:acm:us-east-1:000106928613:certificate/01edc0d4-95b9-476f-8e76-cda2bf3da633"

. "$UTILS_PATH"

REPO_DIR=$(git rev-parse --show-toplevel) || __error-red "Failed to find root of repo."
pushd "$REPO_DIR" > /dev/null

__echo-blue "Querying artifact bucket name..."
S3_BUCKET=$(aws cloudformation list-exports --query 'Exports[?Name==`NewsletterArtifactsBucketName`].Value' --output text) || __error-red 'Failed to get artifact bucket name. Authed with AWS?'
__echo-green "Bucket found with name '$S3_BUCKET'"

__echo-blue "Deleting existing artifacts..."
rm -rf out
__echo-green "Deleted!"

__echo-blue "Building new artifacts..."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/frontend || __error-red "Failed to build frontend package."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/bouncehandler || __error-red "Failed to build bouncehandler package."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/sender || __error-red "Failed to build sender package."
__echo-green "Built!"

__echo-blue "Getting SHA1 of new artifact..."
ARTIFACT_SHASUM=$(sha1sum "$ARTIFACT_PATH"/* | sha1sum | cut -d ' ' -f 1)
__echo-green "Got SHA1 '$ARTIFACT_SHASUM'"

__echo-blue "Zipping new artifact to '$ZIP_PATH'..."
zip -j "$ZIP_PATH" "$ARTIFACT_PATH"/*
__echo-green "Zipped!"

S3_FILE="$ARTIFACT_SHASUM.zip"
S3_PATH="s3://$S3_BUCKET/$S3_FILE"
__echo-blue "Uploading zip '$ZIP_PATH' to S3 path '$S3_PATH'..."
aws s3 cp "$ZIP_PATH" "$S3_PATH" || __error-red "Failed to upload artifact."
__echo-green "Uploaded!"

__echo-blue "Updating stack '$STACK_NAME'..."
HOSTED_ZONE_ID=$(aws route53 list-hosted-zones --query "HostedZones[?Name==\`$HOSTED_ZONE_NAME.\`"].Id --output text | cut -d '/' -f 3)
aws cloudformation update-stack --stack-name "$STACK_NAME" \
  --template-body file://newsletter.cloudformation.yaml \
  --capabilities 'CAPABILITY_IAM' \
  --parameters \
    "ParameterKey=ArtifactPath,ParameterValue=$S3_FILE" \
    "ParameterKey=ArtifactBucket,ParameterValue=$S3_BUCKET" \
    "ParameterKey=CertificateARN,ParameterValue=$CERTIFICATE_ARN" \
    "ParameterKey=DomainName,ParameterValue=$DOMAIN_NAME" \
    "ParameterKey=HostedZoneID,ParameterValue=$HOSTED_ZONE_ID" || 
  __error-red "Failed to update cloudformation"
echo "Waiting for stack update to complete..."
aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME"
__echo-green "Stack updated!"

__echo-blue "Deploying to main stage..."
API_ID=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --query 'Stacks[0].Outputs[?OutputKey==`NewsletterAPIID`].OutputValue' --output text)
aws apigateway create-deployment --stage-name main --rest-api-id "$API_ID" 
__echo-green "Stage deployed!"
