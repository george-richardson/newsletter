#!/bin/zsh
set -e

ARTIFACT_PATH="out/"
ZIP_PATH="out/newsletter.zip"
HOSTED_ZONE_NAME="gjhr.me"

if [[ "$1" == "prod" ]]; then
  echo "Are you sure you want deploy to prod? [Enter to continue]"
  read
  STACK_NAME="newsletter-prod"
  DOMAIN_NAME="newsletter.gjhr.me"
  CERTIFICATE_ARN="arn:aws:acm:us-east-1:000106928613:certificate/0d203eaa-5b1a-4af8-8fe2-c7c1022b1b9e"
else
  STACK_NAME="newsletter-dev"
  DOMAIN_NAME="newsletter-dev.gjhr.me"
  CERTIFICATE_ARN="arn:aws:acm:us-east-1:000106928613:certificate/01edc0d4-95b9-476f-8e76-cda2bf3da633"
fi 

. "$UTILS_PATH"

REPO_DIR=$(git rev-parse --show-toplevel) || _error "Failed to find root of repo."
pushd "$REPO_DIR" > /dev/null

_info "Querying artifact bucket name..."
S3_BUCKET=$(aws cloudformation list-exports --query 'Exports[?Name==`NewsletterArtifactsBucketName`].Value' --output text) || _error 'Failed to get artifact bucket name. Authed with AWS?'
_ok "Bucket found with name '$S3_BUCKET'"

_info "Deleting existing artifacts..."
rm -rf out
_ok "Deleted!"

_info "Building new artifacts..."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/frontend || _error "Failed to build frontend package."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/bouncehandler || _error "Failed to build bouncehandler package."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/sender || _error "Failed to build sender package."
CGO_ENABLED=0 go build -o ./out/ ./lambdas/feedreader || _error "Failed to build feedreader package."
_ok "Built!"

_info "Getting SHA1 of new artifact..."
ARTIFACT_SHASUM=$(sha1sum "$ARTIFACT_PATH"/* | sha1sum | cut -d ' ' -f 1)
_ok "Got SHA1 '$ARTIFACT_SHASUM'"

_info "Zipping new artifact to '$ZIP_PATH'..."
zip -j "$ZIP_PATH" "$ARTIFACT_PATH"/*
_ok "Zipped!"

S3_FILE="$ARTIFACT_SHASUM.zip"
S3_PATH="s3://$S3_BUCKET/$S3_FILE"
_info "Uploading zip '$ZIP_PATH' to S3 path '$S3_PATH'..."
aws s3 cp "$ZIP_PATH" "$S3_PATH" || _error "Failed to upload artifact."
_ok "Uploaded!"

_info "Updating stack '$STACK_NAME'..."
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
  _error "Failed to update cloudformation"
echo "Waiting for stack update to complete..."
aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME"
_ok "Stack updated!"

_info "Deploying to main stage..."
API_ID=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --query 'Stacks[0].Outputs[?OutputKey==`NewsletterAPIID`].OutputValue' --output text)
aws apigateway create-deployment --stage-name main --rest-api-id "$API_ID" 
_ok "Stage deployed!"
