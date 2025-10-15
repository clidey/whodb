#!/bin/bash
#
# Copyright 2025 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <failed_version> <previous_version>"
    echo "Example: $0 0.61.0 0.60.0"
    exit 1
fi

FAILED_VERSION=$1
PREVIOUS_VERSION=$2
IMAGE_NAME="clidey/whodb"

echo "Rolling back Docker images..."
echo "Failed version: $FAILED_VERSION"
echo "Reverting to: $PREVIOUS_VERSION"

# Login to Docker Hub (assumes DOCKERHUB_USERNAME and DOCKERHUB_TOKEN are set)
echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin

# Delete the failed version tags (both version-specific and latest)
echo "Deleting failed version tags..."

# Get Docker Hub token for API calls
TOKEN=$(curl -s -H "Content-Type: application/json" -X POST -d "{\"username\": \"$DOCKERHUB_USERNAME\", \"password\": \"$DOCKERHUB_TOKEN\"}" https://hub.docker.com/v2/users/login/ | jq -r .token)

# Delete the failed version tag
curl -X DELETE -H "Authorization: JWT $TOKEN" "https://hub.docker.com/v2/repositories/${IMAGE_NAME}/tags/${FAILED_VERSION}/" || echo "Warning: Could not delete tag ${FAILED_VERSION}"

# Revert latest tag to previous version
echo "Reverting 'latest' tag to version $PREVIOUS_VERSION..."

# Pull the previous version
docker pull "${IMAGE_NAME}:${PREVIOUS_VERSION}"

# Retag as latest
docker tag "${IMAGE_NAME}:${PREVIOUS_VERSION}" "${IMAGE_NAME}:latest"

# Push the reverted latest tag
docker push "${IMAGE_NAME}:latest"

echo "✓ Rollback complete"
echo "  - Deleted tag: ${FAILED_VERSION}"
echo "  - Reverted 'latest' to: ${PREVIOUS_VERSION}"
