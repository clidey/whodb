#!/bin/bash

# Frontend run script - defaults to CE, use --ee for Enterprise

if [[ "$1" == "--ee" ]]; then
    echo "🏢 Starting Frontend with Enterprise Features..."
    VITE_BUILD_EDITION=ee pnpm run dev
else
    echo "🚀 Starting Frontend (Community Edition)..."
    pnpm run dev
fi