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

#
# App Store Connect API helper script
# Handles JWT authentication and API calls for automated submissions
#
# Required environment variables:
#   APP_STORE_CONNECT_API_KEY_ID - Key ID from App Store Connect
#   APP_STORE_CONNECT_ISSUER_ID  - Issuer ID from App Store Connect
#   APP_STORE_CONNECT_API_KEY    - Private key contents (.p8 file)
#
# Usage:
#   ./appstore-connect.sh submit-version <bundle-id> <version> <build-number> "<release-notes>"
#

set -euo pipefail

API_BASE="https://api.appstoreconnect.apple.com/v1"
JWT_TOKEN=""
JWT_EXPIRY=0

log() {
    echo "[$(date '+%H:%M:%S')] $*"
}

error() {
    echo "[$(date '+%H:%M:%S')] ERROR: $*" >&2
}

# Generate JWT token for App Store Connect API authentication
generate_jwt() {
    local key_id="${APP_STORE_CONNECT_API_KEY_ID:?Missing APP_STORE_CONNECT_API_KEY_ID}"
    local issuer_id="${APP_STORE_CONNECT_ISSUER_ID:?Missing APP_STORE_CONNECT_ISSUER_ID}"
    local private_key="${APP_STORE_CONNECT_API_KEY:?Missing APP_STORE_CONNECT_API_KEY}"

    local now
    now=$(date +%s)

    # Check if current token is still valid (with 60s buffer)
    if [[ -n "$JWT_TOKEN" ]] && [[ $JWT_EXPIRY -gt $((now + 60)) ]]; then
        return 0
    fi

    log "Generating new JWT token..."

    # Use PyJWT which handles ES256 signature format correctly
    # This avoids the complexity of manually converting OpenSSL's DER output to raw R||S format
    local result
    result=$(python3 -c "
import jwt
import time
import sys

try:
    now = int(time.time())
    exp = now + 1200  # 20 minutes

    token = jwt.encode(
        {
            'iss': '$issuer_id',
            'iat': now,
            'exp': exp,
            'aud': 'appstoreconnect-v1'
        },
        '''$private_key''',
        algorithm='ES256',
        headers={'kid': '$key_id', 'typ': 'JWT'}
    )
    print(f'{token}|{exp}')
except Exception as e:
    print(f'ERROR: {e}', file=sys.stderr)
    sys.exit(1)
" 2>&1)

    if [[ $? -ne 0 ]] || [[ "$result" == ERROR:* ]]; then
        error "Failed to generate JWT: $result"
        return 1
    fi

    JWT_TOKEN="${result%|*}"
    JWT_EXPIRY="${result#*|}"

    log "JWT token generated successfully"
}

# Make an API request to App Store Connect
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"

    generate_jwt

    local url="${API_BASE}${endpoint}"
    local curl_args=(
        -s
        -g
        --fail-with-body
        -X "$method"
        -H "Authorization: Bearer $JWT_TOKEN"
        -H "Content-Type: application/json"
    )

    if [[ -n "$data" ]]; then
        curl_args+=(-d "$data")
    fi

    local response
    local http_code
    local curl_exit

    # Capture both response body and HTTP status code
    set +e
    response=$(curl "${curl_args[@]}" -w "\n%{http_code}" "$url" 2>&1)
    curl_exit=$?
    set -e

    http_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')

    # Check for curl errors (connection issues, etc.)
    if [[ $curl_exit -ne 0 ]]; then
        error "Curl failed with exit code $curl_exit"
        error "Endpoint: $method $url"
        error "Response: $response"
        return 1
    fi

    # Validate response is JSON
    if ! echo "$response" | jq -e . > /dev/null 2>&1; then
        error "API returned non-JSON response"
        error "Endpoint: $method $endpoint"
        error "HTTP Code: $http_code"
        error "Response (first 500 chars): ${response:0:500}"
        return 1
    fi

    if [[ "$http_code" -ge 400 ]]; then
        error "API request failed with HTTP $http_code"
        error "Endpoint: $method $endpoint"
        error "Response: $response"
        return 1
    fi

    echo "$response"
}

# Get app ID from bundle identifier
get_app_id() {
    local bundle_id="$1"

    log "Looking up app ID for bundle: $bundle_id"

    local response
    response=$(api_request GET "/apps?filter[bundleId]=$bundle_id")

    local app_id
    app_id=$(echo "$response" | jq -r '.data[0].id // empty')

    if [[ -z "$app_id" ]]; then
        error "App not found for bundle ID: $bundle_id"
        return 1
    fi

    log "Found app ID: $app_id"
    echo "$app_id"
}

# Get or create an app store version
get_or_create_version() {
    local app_id="$1"
    local version="$2"

    log "Checking for existing version: $version"

    # Look for existing editable version
    local response
    response=$(api_request GET "/apps/$app_id/appStoreVersions?filter[versionString]=$version&filter[platform]=MAC_OS")

    local version_id
    version_id=$(echo "$response" | jq -r '.data[0].id // empty')

    if [[ -n "$version_id" ]]; then
        local state
        state=$(echo "$response" | jq -r '.data[0].attributes.appStoreState')
        log "Found existing version $version (ID: $version_id, state: $state)"

        # Check if version is in an editable state
        case "$state" in
            PREPARE_FOR_SUBMISSION|DEVELOPER_REJECTED|REJECTED|METADATA_REJECTED|INVALID_BINARY)
                log "Version is editable"
                echo "$version_id"
                return 0
                ;;
            *)
                error "Version $version exists but is in non-editable state: $state"
                return 1
                ;;
        esac
    fi

    log "Creating new version: $version"

    local create_data
    create_data=$(cat <<EOF
{
    "data": {
        "type": "appStoreVersions",
        "attributes": {
            "versionString": "$version",
            "platform": "MAC_OS",
            "releaseType": "AFTER_APPROVAL"
        },
        "relationships": {
            "app": {
                "data": {
                    "type": "apps",
                    "id": "$app_id"
                }
            }
        }
    }
}
EOF
)

    response=$(api_request POST "/appStoreVersions" "$create_data")
    version_id=$(echo "$response" | jq -r '.data.id // empty')

    if [[ -z "$version_id" ]]; then
        error "Failed to create version"
        error "Response: $response"
        return 1
    fi

    log "Created version ID: $version_id"
    echo "$version_id"
}

# Wait for build to be processed and available
wait_for_build() {
    local app_id="$1"
    local version="$2"
    local max_attempts="${3:-60}"  # Default 60 attempts (30 minutes with 30s intervals)

    log "Waiting for build version $version to be processed..."

    local attempt=0
    while [[ $attempt -lt $max_attempts ]]; do
        attempt=$((attempt + 1))

        local response
        response=$(api_request GET "/builds?filter[app]=$app_id&filter[version]=$version&filter[processingState]=VALID&limit=1")

        local build_id
        build_id=$(echo "$response" | jq -r '.data[0].id // empty')

        if [[ -n "$build_id" ]]; then
            log "Build ready (ID: $build_id)"
            echo "$build_id"
            return 0
        fi

        # Check if build exists but is still processing
        response=$(api_request GET "/builds?filter[app]=$app_id&filter[version]=$version&limit=1")
        local processing_state
        processing_state=$(echo "$response" | jq -r '.data[0].attributes.processingState // empty')

        if [[ -n "$processing_state" ]]; then
            log "Build processing state: $processing_state (attempt $attempt/$max_attempts)"
        else
            log "Build not yet visible in App Store Connect (attempt $attempt/$max_attempts)"
        fi

        sleep 30
    done

    error "Timeout waiting for build to be processed"
    return 1
}

# Associate a build with a version
set_version_build() {
    local version_id="$1"
    local build_id="$2"

    log "Associating build $build_id with version $version_id"

    local data
    data=$(cat <<EOF
{
    "data": {
        "type": "builds",
        "id": "$build_id"
    }
}
EOF
)

    api_request PATCH "/appStoreVersions/$version_id/relationships/build" "$data" > /dev/null
    log "Build associated successfully"
}

# Update the "What's New" release notes for a version
set_release_notes() {
    local version_id="$1"
    local release_notes="$2"

    log "Setting release notes..."

    # Get the localization ID for en-US (or create if needed)
    local response
    response=$(api_request GET "/appStoreVersions/$version_id/appStoreVersionLocalizations")

    local localization_id
    localization_id=$(echo "$response" | jq -r '.data[] | select(.attributes.locale == "en-US") | .id' | head -1)

    if [[ -z "$localization_id" ]]; then
        log "Creating en-US localization..."

        local create_data
        create_data=$(cat <<EOF
{
    "data": {
        "type": "appStoreVersionLocalizations",
        "attributes": {
            "locale": "en-US",
            "whatsNew": $(echo "$release_notes" | jq -Rs .)
        },
        "relationships": {
            "appStoreVersion": {
                "data": {
                    "type": "appStoreVersions",
                    "id": "$version_id"
                }
            }
        }
    }
}
EOF
)

        response=$(api_request POST "/appStoreVersionLocalizations" "$create_data")
        localization_id=$(echo "$response" | jq -r '.data.id // empty')

        if [[ -z "$localization_id" ]]; then
            error "Failed to create localization"
            return 1
        fi
    else
        log "Updating existing en-US localization..."

        local update_data
        update_data=$(cat <<EOF
{
    "data": {
        "type": "appStoreVersionLocalizations",
        "id": "$localization_id",
        "attributes": {
            "whatsNew": $(echo "$release_notes" | jq -Rs .)
        }
    }
}
EOF
)

        api_request PATCH "/appStoreVersionLocalizations/$localization_id" "$update_data" > /dev/null
    fi

    log "Release notes updated successfully"
}

# Submit the version for App Store review
submit_for_review() {
    local version_id="$1"

    log "Submitting version for App Store review..."

    local data
    data=$(cat <<EOF
{
    "data": {
        "type": "appStoreVersionSubmissions",
        "relationships": {
            "appStoreVersion": {
                "data": {
                    "type": "appStoreVersions",
                    "id": "$version_id"
                }
            }
        }
    }
}
EOF
)

    local response
    response=$(api_request POST "/appStoreVersionSubmissions" "$data")

    local submission_id
    submission_id=$(echo "$response" | jq -r '.data.id // empty')

    if [[ -z "$submission_id" ]]; then
        # Check if already submitted
        if echo "$response" | jq -e '.errors[].code == "ENTITY_ERROR.RELATIONSHIP.INVALID"' > /dev/null 2>&1; then
            log "Version appears to already be submitted"
            return 0
        fi
        error "Failed to submit for review"
        error "Response: $response"
        return 1
    fi

    log "Submitted for review (submission ID: $submission_id)"
}

# Main command: submit a new version
cmd_submit_version() {
    local bundle_id="$1"
    local version="$2"
    local build_number="$3"
    local release_notes="$4"

    echo "═══════════════════════════════════════════════════════════"
    echo "  App Store Connect - Automated Submission"
    echo "═══════════════════════════════════════════════════════════"
    echo ""
    echo "Bundle ID:     $bundle_id"
    echo "Version:       $version"
    echo "Build Number:  $build_number"
    echo ""

    # Step 1: Get app ID
    local app_id
    app_id=$(get_app_id "$bundle_id")

    # Step 2: Get or create version
    local version_id
    version_id=$(get_or_create_version "$app_id" "$version")

    # Step 3: Wait for build to be processed
    local build_id
    build_id=$(wait_for_build "$app_id" "$build_number")

    # Step 4: Associate build with version
    set_version_build "$version_id" "$build_id"

    # Step 5: Set release notes
    set_release_notes "$version_id" "$release_notes"

    # Step 6: Submit for review
    submit_for_review "$version_id"

    echo ""
    echo "═══════════════════════════════════════════════════════════"
    echo "  Submission Complete"
    echo "═══════════════════════════════════════════════════════════"
    echo ""
    echo "The app has been submitted for App Store review."
    echo "Apple typically reviews apps within 24-48 hours."
    echo ""
    echo "Monitor status at: https://appstoreconnect.apple.com/apps"
    echo ""
}

# Main entry point
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        submit-version)
            if [[ $# -lt 4 ]]; then
                error "Usage: $0 submit-version <bundle-id> <version> <build-number> <release-notes>"
                exit 1
            fi
            cmd_submit_version "$@"
            ;;
        help|--help|-h)
            echo "App Store Connect API Helper"
            echo ""
            echo "Commands:"
            echo "  submit-version <bundle-id> <version> <build-number> <release-notes>"
            echo "      Create/update version, set release notes, and submit for review"
            echo ""
            echo "Required environment variables:"
            echo "  APP_STORE_CONNECT_API_KEY_ID  - Key ID from App Store Connect"
            echo "  APP_STORE_CONNECT_ISSUER_ID   - Issuer ID from App Store Connect"
            echo "  APP_STORE_CONNECT_API_KEY     - Private key contents (.p8 file)"
            ;;
        *)
            error "Unknown command: $command"
            exit 1
            ;;
    esac
}

main "$@"