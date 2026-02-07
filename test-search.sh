#!/bin/bash
# Test script for diary search TUI
#
# This script demonstrates the interactive search feature
# Run this in your terminal (not via Claude Code) to test the TUI

set -e

echo "=== Diary Search TUI Test ==="
echo
echo "This will search testuser2's diary for 'encryption'"
echo "Press any key to continue..."
read -n 1 -s
echo

cd "$(dirname "$0")"

# Build if needed
if [ ! -f "./diary" ]; then
    echo "Building diary binary..."
    go build -o diary ./cmd/diary/
fi

# Run search
echo "Launching search TUI..."
./diary testuser2 search "encryption"

echo
echo "=== Test complete ==="
