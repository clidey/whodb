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

# This script captures the current state of store deployments
# to enable better rollback capabilities

MANIFEST_FILE="store_state.json"
SNAP_NAME=${1:-"whodb"}

echo "Capturing current store states for potential rollback..."

# Initialize the state file
echo "{
  \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\",
  \"snap\": {},
  \"windows_store\": {},
  \"mac_store\": {}
}" > "$MANIFEST_FILE"

# ============================================
# Snap Store State
# ============================================
if [ -n "$SNAPCRAFT_STORE_CREDENTIALS" ]; then
    echo "Querying Snap Store state..."

    # Get current revisions for stable channel
    # snapcraft list-revisions returns JSON with revision info
    SNAP_INFO=$(snapcraft list-revisions "$SNAP_NAME" --format json 2>/dev/null || echo "{}")

    if [ "$SNAP_INFO" != "{}" ]; then
        # Extract the current stable channel revision
        STABLE_REVISION=$(echo "$SNAP_INFO" | jq -r '.[] | select(.channels | contains(["stable"])) | .revision' | head -1)
        STABLE_VERSION=$(echo "$SNAP_INFO" | jq -r '.[] | select(.channels | contains(["stable"])) | .version' | head -1)

        # Get total number of revisions
        TOTAL_REVISIONS=$(echo "$SNAP_INFO" | jq '. | length')

        # Store the last 3 revision numbers for potential rollback
        RECENT_REVISIONS=$(echo "$SNAP_INFO" | jq -r '.[0:3] | [.[] | {revision: .revision, version: .version, architectures: .architectures}]')

        # Update manifest
        jq ".snap = {
            \"current_stable_revision\": \"$STABLE_REVISION\",
            \"current_stable_version\": \"$STABLE_VERSION\",
            \"total_revisions\": $TOTAL_REVISIONS,
            \"recent_revisions\": $RECENT_REVISIONS,
            \"captured_at\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
        }" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"

        echo "✓ Snap Store: Current stable revision $STABLE_REVISION (version $STABLE_VERSION)"
    else
        echo "⚠ No Snap Store information available (may not be published yet)"
        jq '.snap.status = "not_published"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
    fi
else
    echo "⚠ Snap Store credentials not available"
    jq '.snap.status = "no_credentials"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
fi

# ============================================
# Windows Store State
# ============================================
if [ -n "$WINDOWS_TENANT_ID" ] && [ -n "$WINDOWS_CLIENT_ID" ] && [ -n "$WINDOWS_CLIENT_SECRET" ]; then
    echo "Querying Windows Store state..."

    # Get access token for Partner Center API
    ACCESS_TOKEN=$(curl -s -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "grant_type=client_credentials" \
        -d "client_id=$WINDOWS_CLIENT_ID" \
        -d "client_secret=$WINDOWS_CLIENT_SECRET" \
        -d "resource=https://manage.devcenter.microsoft.com" \
        "https://login.microsoftonline.com/$WINDOWS_TENANT_ID/oauth2/token" | jq -r .access_token)

    if [ "$ACCESS_TOKEN" != "null" ] && [ -n "$ACCESS_TOKEN" ]; then
        # Get app ID (you'll need to set WINDOWS_APP_ID in secrets)
        APP_ID=${WINDOWS_APP_ID:-""}

        if [ -n "$APP_ID" ]; then
            # Get current submission info
            SUBMISSION_INFO=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                "https://manage.devcenter.microsoft.com/v1.0/my/applications/$APP_ID/submissions" || echo "{}")

            # Check for pending submission
            PENDING_SUBMISSION=$(echo "$SUBMISSION_INFO" | jq -r '.pendingApplicationSubmission.id // "none"')
            PUBLISHED_VERSION=$(echo "$SUBMISSION_INFO" | jq -r '.lastPublishedApplicationSubmission.applicationPackages[0].version // "unknown"')

            jq ".windows_store = {
                \"app_id\": \"$APP_ID\",
                \"pending_submission\": \"$PENDING_SUBMISSION\",
                \"published_version\": \"$PUBLISHED_VERSION\",
                \"captured_at\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
            }" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"

            echo "✓ Windows Store: Published version $PUBLISHED_VERSION, Pending submission: $PENDING_SUBMISSION"
        else
            echo "⚠ Windows Store app ID not configured"
            jq '.windows_store.status = "no_app_id"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
        fi
    else
        echo "⚠ Could not get Windows Store access token"
        jq '.windows_store.status = "auth_failed"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
    fi
else
    echo "⚠ Windows Store credentials not available"
    jq '.windows_store.status = "no_credentials"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
fi

# ============================================
# Mac App Store State
# ============================================
if [ -n "$APPLE_TEAM_ID" ] && [ -n "$APPLE_KEY_ID" ] && [ -n "$APPLE_KEY_CONTENT" ]; then
    echo "Querying Mac App Store state..."

    # Generate JWT for App Store Connect API
    # Note: This requires a proper JWT library or implementation
    # For now, we'll use a placeholder

    APP_ID=${MAC_APP_ID:-""}

    if [ -n "$APP_ID" ]; then
        # This would require proper JWT signing and API calls
        # Placeholder for Mac App Store state
        jq ".mac_store = {
            \"app_id\": \"$APP_ID\",
            \"status\": \"requires_implementation\",
            \"note\": \"JWT signing required for App Store Connect API\",
            \"captured_at\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
        }" "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"

        echo "⚠ Mac App Store: API integration pending implementation"
    else
        echo "⚠ Mac App Store app ID not configured"
        jq '.mac_store.status = "no_app_id"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
    fi
else
    echo "⚠ Mac App Store credentials not available"
    jq '.mac_store.status = "no_credentials"' "$MANIFEST_FILE" > tmp.json && mv tmp.json "$MANIFEST_FILE"
fi

echo ""
echo "Store state captured in: $MANIFEST_FILE"
echo ""
echo "Summary:"
jq . "$MANIFEST_FILE"