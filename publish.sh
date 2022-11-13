#!/bin/zsh
set -e

ARTIFACT_PATH="out/newsletter"
ZIP_PATH="out/newsletter.zip"
STACK_NAME="newsletter"

. "$UTILS_PATH"

REPO_DIR=$(git rev-parse --show-toplevel) || __error-red "Failed to find root of repo."
pushd "$REPO_DIR" > /dev/null

__echo-blue "Querying artifact bucket name..."
S3_BUCKET=$(aws cloudformation list-exports --query 'Exports[?Name==`NewsletterArtifactsBucketName`].Value' --output text) || __error-red 'Failed to get artifact bucket name. Authed with AWS?'
__echo-green "Bucket found with name '$S3_BUCKET'"

__echo-blue "Deleting existing artifacts..."
rm out/*
__echo-green "Deleted!"

__echo-blue "Building new artifacts..."
go build -o out/ || __error-red "Failed to build package."
__echo-green "Built!"

__echo-blue "Getting SHA1 of new artifact..."
ARTIFACT_SHASUM=$(sha1sum "$ARTIFACT_PATH" | cut -d ' ' -f 1)
__echo-green "Got SHA1 '$ARTIFACT_SHASUM'"

__echo-blue "Zipping new artifact to '$ZIP_PATH'..."
zip "$ZIP_PATH" "$ARTIFACT_PATH"
__echo-green "Zipped!"

S3_FILE="$ARTIFACT_SHASUM.zip"
S3_PATH="s3://$S3_BUCKET/$S3_FILE"
__echo-blue "Uploading zip '$ZIP_PATH' to S3 path '$S3_PATH'..."
aws s3 cp "$ZIP_PATH" "$S3_PATH" || __error-red "Failed to upload artifact."
__echo-green "Uploaded!"

__echo-blue "Updating stack '$STACK_NAME'..."
aws cloudformation update-stack --stack-name "$STACK_NAME" \
  --template-body file://newsletter.cloudformation.yaml \
  --capabilities 'CAPABILITY_IAM' \
  --parameters \
    "ParameterKey=ArtifactPath,ParameterValue=$S3_FILE" \
    "ParameterKey=ArtifactBucket,ParameterValue=$S3_BUCKET" || 
  __error-red "Failed to update cloudformation"
echo "Waiting for stack update to complete..."
aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME"
__echo-green "Stack updated!"
