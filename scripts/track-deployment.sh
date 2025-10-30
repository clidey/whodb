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

# Usage: track-deployment.sh <service> <status> [version]
# Example: track-deployment.sh docker deployed 0.61.0

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <service> <status> [version]"
    echo "Services: docker, snap, github_release, windows_store, mac_store"
    echo "Status: pending, deployed, failed"
    exit 1
fi

SERVICE=$1
STATUS=$2
VERSION=${3:-"unknown"}
MANIFEST_FILE="deployment_manifest.json"

# Initialize manifest if it doesn't exist
if [ ! -f "$MANIFEST_FILE" ]; then
    echo "{
  \"version\": \"$VERSION\",
  \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\",
  \"docker\": {
    \"deployed\": false,
    \"timestamp\": null,
    \"status\": \"pending\"
  },
  \"snap\": {
    \"deployed\": false,
    \"timestamp\": null,
    \"status\": \"pending\"
  },
  \"github_release\": {
    \"created\": false,
    \"timestamp\": null,
    \"status\": \"pending\",
    \"release_id\": null
  },
  \"windows_store\": {
    \"submitted\": false,
    \"timestamp\": null,
    \"status\": \"pending\",
    \"submission_id\": null
  },
  \"mac_store\": {
    \"submitted\": false,
    \"timestamp\": null,
    \"status\": \"pending\",
    \"submission_id\": null
  }
}" > "$MANIFEST_FILE"
fi

# Update the manifest based on service and status
case "$SERVICE" in
    docker)
        if [ "$STATUS" = "deployed" ]; then
            jq ".docker.deployed = true | .docker.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\" | .docker.status = \"deployed\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✓ Docker deployment tracked"
        elif [ "$STATUS" = "failed" ]; then
            jq ".docker.status = \"failed\" | .docker.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✗ Docker deployment failed"
        fi
        ;;

    snap)
        if [ "$STATUS" = "deployed" ]; then
            jq ".snap.deployed = true | .snap.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\" | .snap.status = \"deployed\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✓ Snap deployment tracked"
        elif [ "$STATUS" = "failed" ]; then
            jq ".snap.status = \"failed\" | .snap.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✗ Snap deployment failed"
        fi
        ;;

    github_release)
        if [ "$STATUS" = "deployed" ]; then
            RELEASE_ID=${4:-null}
            jq ".github_release.created = true | .github_release.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\" | .github_release.status = \"deployed\" | .github_release.release_id = \"$RELEASE_ID\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✓ GitHub release tracked"
        elif [ "$STATUS" = "failed" ]; then
            jq ".github_release.status = \"failed\" | .github_release.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✗ GitHub release failed"
        fi
        ;;

    windows_store)
        if [ "$STATUS" = "deployed" ]; then
            SUBMISSION_ID=${4:-null}
            jq ".windows_store.submitted = true | .windows_store.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\" | .windows_store.status = \"deployed\" | .windows_store.submission_id = \"$SUBMISSION_ID\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✓ Windows Store submission tracked"
        elif [ "$STATUS" = "failed" ]; then
            jq ".windows_store.status = \"failed\" | .windows_store.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✗ Windows Store submission failed"
        fi
        ;;

    mac_store)
        if [ "$STATUS" = "deployed" ]; then
            SUBMISSION_ID=${4:-null}
            jq ".mac_store.submitted = true | .mac_store.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\" | .mac_store.status = \"deployed\" | .mac_store.submission_id = \"$SUBMISSION_ID\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✓ Mac Store submission tracked"
        elif [ "$STATUS" = "failed" ]; then
            jq ".mac_store.status = \"failed\" | .mac_store.timestamp = \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
            echo "✗ Mac Store submission failed"
        fi
        ;;

    *)
        echo "Unknown service: $SERVICE"
        exit 1
        ;;
esac

# Output current status
echo "Current deployment status:"
jq . "$MANIFEST_FILE"