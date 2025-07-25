#!/bin/bash

# Simple run script - defaults to CE, use --ee for Enterprise

if [[ "$1" == "--ee" ]]; then
    echo "🏢 Starting WhoDB Enterprise Edition..."
    cd core && go run -tags ee .
else
    echo "🚀 Starting WhoDB Community Edition..."
    cd core && go run .
fi