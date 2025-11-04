# WhoDB Enterprise Edition Desktop Makefile
# This file sets edition-specific variables and includes the common build logic

# Edition-specific configuration
EDITION := ee
EDITION_NAME := EE
PKG_ID := com.clidey.whodb.ee
GOWORK := go.work.desktop-ee
BUILD_TAGS := -tags ee
OUTPUT_PREFIX := whodb-ee
FRONTEND_BUILD_CMD := build:ee
APP_NAME := WhoDB - Enterprise.app
DMG_VOLNAME := WhoDB - Enterprise
LINUX_TAGS := -tags "ee webkit2_41"

# Include common build logic
include ../desktop-common/Makefile.common