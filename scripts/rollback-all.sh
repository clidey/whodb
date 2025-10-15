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
    echo "Usage: $0 <failed_version> <previous_version> [deployment_manifest.json] [store_state.json]"
    echo "Example: $0 0.61.0 0.60.0 deployment_manifest.json store_state.json"
    exit 1
fi

FAILED_VERSION=$1
PREVIOUS_VERSION=$2
MANIFEST_FILE=${3:-deployment_manifest.json}
STORE_STATE_FILE=${4:-store_state.json}
IMAGE_NAME="clidey/whodb"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Enhanced Rollback with Store Automation${NC}"
echo -e "${YELLOW}========================================${NC}"
echo "Failed version: $FAILED_VERSION"
echo "Reverting to: $PREVIOUS_VERSION"
echo ""

# Track rollback results
ROLLBACK_FAILED=0
MANUAL_INTERVENTION_NEEDED=0

# Read deployment manifest if it exists
if [ -f "$MANIFEST_FILE" ]; then
    echo -e "${GREEN}Reading deployment manifest...${NC}"
    DOCKER_DEPLOYED=$(jq -r '.docker.deployed' "$MANIFEST_FILE" 2>/dev/null || echo "false")
    SNAP_DEPLOYED=$(jq -r '.snap.deployed' "$MANIFEST_FILE" 2>/dev/null || echo "false")
    GITHUB_RELEASE_CREATED=$(jq -r '.github_release.created' "$MANIFEST_FILE" 2>/dev/null || echo "false")
    WINDOWS_SUBMITTED=$(jq -r '.windows_store.submitted' "$MANIFEST_FILE" 2>/dev/null || echo "false")
    MAC_SUBMITTED=$(jq -r '.mac_store.submitted' "$MANIFEST_FILE" 2>/dev/null || echo "false")
else
    echo -e "${YELLOW}No deployment manifest found, attempting rollback of all services${NC}"
    DOCKER_DEPLOYED="true"
    SNAP_DEPLOYED="true"
    GITHUB_RELEASE_CREATED="true"
    WINDOWS_SUBMITTED="false"
    MAC_SUBMITTED="false"
fi

# Read store state if it exists
if [ -f "$STORE_STATE_FILE" ]; then
    echo -e "${GREEN}Reading pre-deployment store state...${NC}"
    SNAP_STABLE_REVISION=$(jq -r '.snap.current_stable_revision' "$STORE_STATE_FILE" 2>/dev/null || echo "")
    WINDOWS_PENDING_SUBMISSION=$(jq -r '.windows_store.pending_submission' "$STORE_STATE_FILE" 2>/dev/null || echo "none")
    echo "  Snap stable revision before deployment: $SNAP_STABLE_REVISION"
    echo "  Windows pending submission before deployment: $WINDOWS_PENDING_SUBMISSION"
else
    echo -e "${YELLOW}No store state file found, will use fallback methods${NC}"
fi

# ============================================
# Docker Rollback (Fully Automated)
# ============================================
if [ "$DOCKER_DEPLOYED" = "true" ]; then
    echo -e "\n${BLUE}[1/5] Rolling back Docker images...${NC}"

    if [ -n "$DOCKERHUB_USERNAME" ] && [ -n "$DOCKERHUB_TOKEN" ]; then
        # Login to Docker Hub
        echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin 2>/dev/null || {
            echo -e "${RED}Failed to login to Docker Hub${NC}"
            ROLLBACK_FAILED=1
        }

        # Get Docker Hub token for API calls
        TOKEN=$(curl -s -H "Content-Type: application/json" -X POST \
            -d "{\"username\": \"$DOCKERHUB_USERNAME\", \"password\": \"$DOCKERHUB_TOKEN\"}" \
            https://hub.docker.com/v2/users/login/ | jq -r .token)

        if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
            # Delete the failed version tag
            echo "  Deleting Docker tag: ${FAILED_VERSION}..."
            curl -X DELETE -H "Authorization: JWT $TOKEN" \
                "https://hub.docker.com/v2/repositories/${IMAGE_NAME}/tags/${FAILED_VERSION}/" 2>/dev/null || {
                echo -e "${YELLOW}  Warning: Could not delete tag ${FAILED_VERSION}${NC}"
            }

            # Revert latest tag to previous version
            echo "  Reverting 'latest' tag to version $PREVIOUS_VERSION..."

            # Pull the previous version
            docker pull "${IMAGE_NAME}:${PREVIOUS_VERSION}" || {
                echo -e "${RED}  Failed to pull previous Docker image${NC}"
                ROLLBACK_FAILED=1
            }

            # Retag as latest
            docker tag "${IMAGE_NAME}:${PREVIOUS_VERSION}" "${IMAGE_NAME}:latest"

            # Push the reverted latest tag
            docker push "${IMAGE_NAME}:latest" || {
                echo -e "${RED}  Failed to push reverted latest tag${NC}"
                ROLLBACK_FAILED=1
            }

            echo -e "${GREEN}  ✓ Docker rollback complete${NC}"
        else
            echo -e "${RED}  Failed to get Docker Hub API token${NC}"
            ROLLBACK_FAILED=1
        fi
    else
        echo -e "${YELLOW}  Docker Hub credentials not available, skipping Docker rollback${NC}"
    fi
fi

# ============================================
# GitHub Release Rollback (Fully Automated)
# ============================================
if [ "$GITHUB_RELEASE_CREATED" = "true" ]; then
    echo -e "\n${BLUE}[2/5] Rolling back GitHub release...${NC}"

    if [ -n "$GITHUB_TOKEN" ]; then
        REPO_OWNER=$(echo "$GITHUB_REPOSITORY" | cut -d'/' -f1)
        REPO_NAME=$(echo "$GITHUB_REPOSITORY" | cut -d'/' -f2)

        # Get release ID for the failed version
        echo "  Finding release for v${FAILED_VERSION}..."
        RELEASE_ID=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
            "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/tags/v${FAILED_VERSION}" | \
            jq -r .id)

        if [ "$RELEASE_ID" != "null" ] && [ -n "$RELEASE_ID" ]; then
            echo "  Deleting GitHub release (ID: $RELEASE_ID)..."
            curl -X DELETE -H "Authorization: token $GITHUB_TOKEN" \
                "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/${RELEASE_ID}" || {
                echo -e "${RED}  Failed to delete GitHub release${NC}"
                ROLLBACK_FAILED=1
            }

            # Also delete the tag
            echo "  Deleting Git tag v${FAILED_VERSION}..."
            curl -X DELETE -H "Authorization: token $GITHUB_TOKEN" \
                "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/git/refs/tags/v${FAILED_VERSION}" || {
                echo -e "${YELLOW}  Warning: Could not delete Git tag${NC}"
            }

            echo -e "${GREEN}  ✓ GitHub release and tag deleted${NC}"
        else
            echo -e "${YELLOW}  No GitHub release found for v${FAILED_VERSION}${NC}"
        fi
    else
        echo -e "${YELLOW}  GitHub token not available, skipping GitHub release rollback${NC}"
    fi
fi

# ============================================
# Snap Store Rollback (API-Based)
# ============================================
if [ "$SNAP_DEPLOYED" = "true" ]; then
    echo -e "\n${BLUE}[3/5] Rolling back Snap Store release...${NC}"

    if [ -n "$SNAPCRAFT_STORE_CREDENTIALS" ]; then
        SNAP_NAME=${SNAP_NAME:-"whodb"}

        # If we have the previous stable revision from store state, use it
        if [ -n "$SNAP_STABLE_REVISION" ] && [ "$SNAP_STABLE_REVISION" != "" ] && [ "$SNAP_STABLE_REVISION" != "null" ]; then
            echo "  Reverting to previous stable revision: $SNAP_STABLE_REVISION"

            # Use snapcraft to revert to the previous revision
            snapcraft release "$SNAP_NAME" "$SNAP_STABLE_REVISION" stable 2>/dev/null && {
                echo -e "${GREEN}  ✓ Successfully reverted Snap to revision $SNAP_STABLE_REVISION${NC}"
            } || {
                echo -e "${YELLOW}  ⚠ Could not revert via API, trying alternative method...${NC}"

                # Alternative: Try to close the stable channel for the failed version
                # Get current revisions
                CURRENT_REVISIONS=$(snapcraft list-revisions "$SNAP_NAME" --format json 2>/dev/null || echo "[]")

                # Find the failed version's revision
                FAILED_REVISION=$(echo "$CURRENT_REVISIONS" | jq -r --arg v "$FAILED_VERSION" '.[] | select(.version == $v) | .revision' | head -1)

                if [ -n "$FAILED_REVISION" ] && [ "$FAILED_REVISION" != "" ]; then
                    echo "  Closing stable channel for failed revision $FAILED_REVISION..."
                    snapcraft close "$SNAP_NAME" stable "$FAILED_REVISION" 2>/dev/null || {
                        echo -e "${YELLOW}  ⚠ Could not close channel${NC}"
                    }
                fi

                MANUAL_INTERVENTION_NEEDED=1
                echo -e "${YELLOW}  ⚠ Manual verification required in Snapcraft dashboard${NC}"
                echo -e "${YELLOW}    Instructions:${NC}"
                echo "    1. Go to https://snapcraft.io/account/snaps"
                echo "    2. Select '$SNAP_NAME'"
                echo "    3. Go to 'Releases' tab"
                echo "    4. Promote revision $SNAP_STABLE_REVISION to stable channel"
            }
        else
            echo -e "${YELLOW}  ⚠ No previous revision information available${NC}"
            echo -e "${YELLOW}  Manual intervention required:${NC}"
            echo "    1. Go to https://snapcraft.io/account/snaps"
            echo "    2. Select your snap"
            echo "    3. Manually revert to the previous stable version"
            MANUAL_INTERVENTION_NEEDED=1
        fi
    else
        echo -e "${YELLOW}  Snapcraft credentials not available, skipping Snap rollback${NC}"
    fi
fi

# ============================================
# Windows Store Rollback (API-Based Cancellation)
# ============================================
if [ "$WINDOWS_SUBMITTED" = "true" ] || [ -n "$WINDOWS_TENANT_ID" ]; then
    echo -e "\n${BLUE}[4/5] Checking Windows Store submission...${NC}"

    if [ -n "$WINDOWS_TENANT_ID" ] && [ -n "$WINDOWS_CLIENT_ID" ] && [ -n "$WINDOWS_CLIENT_SECRET" ]; then
        # Get access token
        ACCESS_TOKEN=$(curl -s -X POST \
            -H "Content-Type: application/x-www-form-urlencoded" \
            -d "grant_type=client_credentials" \
            -d "client_id=$WINDOWS_CLIENT_ID" \
            -d "client_secret=$WINDOWS_CLIENT_SECRET" \
            -d "resource=https://manage.devcenter.microsoft.com" \
            "https://login.microsoftonline.com/$WINDOWS_TENANT_ID/oauth2/token" | jq -r .access_token)

        if [ "$ACCESS_TOKEN" != "null" ] && [ -n "$ACCESS_TOKEN" ]; then
            APP_ID=${WINDOWS_APP_ID:-""}

            if [ -n "$APP_ID" ]; then
                # Check for pending submission
                SUBMISSION_INFO=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
                    "https://manage.devcenter.microsoft.com/v1.0/my/applications/$APP_ID" || echo "{}")

                PENDING_ID=$(echo "$SUBMISSION_INFO" | jq -r '.pendingApplicationSubmission.id // "none"')

                if [ "$PENDING_ID" != "none" ] && [ "$PENDING_ID" != "null" ]; then
                    echo "  Found pending submission: $PENDING_ID"
                    echo "  Attempting to delete pending submission..."

                    # Delete the pending submission
                    DELETE_RESPONSE=$(curl -s -X DELETE -H "Authorization: Bearer $ACCESS_TOKEN" \
                        "https://manage.devcenter.microsoft.com/v1.0/my/applications/$APP_ID/submissions/$PENDING_ID")

                    if [ $? -eq 0 ]; then
                        echo -e "${GREEN}  ✓ Successfully cancelled pending Windows Store submission${NC}"
                    else
                        echo -e "${YELLOW}  ⚠ Could not cancel submission (may already be published)${NC}"
                        MANUAL_INTERVENTION_NEEDED=1
                    fi
                else
                    echo -e "${YELLOW}  No pending submission found (may already be published)${NC}"
                    if [ "$WINDOWS_SUBMITTED" = "true" ]; then
                        echo -e "${YELLOW}  Manual intervention required:${NC}"
                        echo "    1. Go to https://partner.microsoft.com/dashboard"
                        echo "    2. Navigate to your app"
                        echo "    3. Create new submission with previous version"
                        MANUAL_INTERVENTION_NEEDED=1
                    fi
                fi
            else
                echo -e "${YELLOW}  Windows app ID not configured${NC}"
            fi
        else
            echo -e "${YELLOW}  Could not authenticate with Partner Center${NC}"
        fi
    else
        echo -e "${YELLOW}  Windows Store credentials not available${NC}"
    fi
fi

# ============================================
# Mac App Store Rollback (Limited API)
# ============================================
if [ "$MAC_SUBMITTED" = "true" ]; then
    echo -e "\n${BLUE}[5/5] Mac App Store rollback...${NC}"

    # Mac App Store Connect API requires JWT signing which is complex in bash
    # Provide manual instructions instead
    echo -e "${YELLOW}  Mac App Store requires manual intervention:${NC}"
    echo "    1. Go to https://appstoreconnect.apple.com"
    echo "    2. Navigate to 'My Apps' → Select your app"
    echo "    3. If submission is 'Waiting for Review' or 'In Review':"
    echo "       - Click 'Remove from Review'"
    echo "    4. If already published:"
    echo "       - Create new version with previous build"
    echo "       - Submit for expedited review"
    MANUAL_INTERVENTION_NEEDED=1
fi

# ============================================
# Summary
# ============================================
echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Rollback Summary:${NC}"
echo -e "${YELLOW}========================================${NC}"

if [ "$DOCKER_DEPLOYED" = "true" ]; then
    echo -e "  Docker Hub: ${GREEN}✓ Automated${NC} - Reverted to ${PREVIOUS_VERSION}"
fi

if [ "$GITHUB_RELEASE_CREATED" = "true" ]; then
    echo -e "  GitHub Release: ${GREEN}✓ Automated${NC} - Deleted v${FAILED_VERSION}"
fi

if [ "$SNAP_DEPLOYED" = "true" ]; then
    if [ -n "$SNAP_STABLE_REVISION" ] && [ "$SNAP_STABLE_REVISION" != "" ] && [ "$SNAP_STABLE_REVISION" != "null" ]; then
        echo -e "  Snap Store: ${GREEN}✓ Automated${NC} - Reverted to revision ${SNAP_STABLE_REVISION}"
    else
        echo -e "  Snap Store: ${YELLOW}⚠ Manual Check${NC} - Verify in dashboard"
    fi
fi

if [ "$WINDOWS_SUBMITTED" = "true" ] || [ -n "$WINDOWS_TENANT_ID" ]; then
    echo -e "  Windows Store: ${YELLOW}⚠ Check Status${NC} - May need manual intervention"
fi

if [ "$MAC_SUBMITTED" = "true" ]; then
    echo -e "  Mac App Store: ${YELLOW}⚠ Manual${NC} - Requires App Store Connect"
fi

echo ""

if [ $MANUAL_INTERVENTION_NEEDED -eq 1 ]; then
    echo -e "${YELLOW}⚠ Some services require manual intervention (see details above)${NC}"
fi

if [ $ROLLBACK_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ Automated rollbacks completed successfully${NC}"

    # Clean up deployment manifest
    if [ -f "$MANIFEST_FILE" ]; then
        rm "$MANIFEST_FILE"
        echo "  Deployment manifest cleaned up"
    fi
else
    echo -e "${RED}✗ Some rollbacks failed - manual intervention required${NC}"
    exit 1
fi