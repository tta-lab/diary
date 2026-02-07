#!/bin/bash
# Demo script for diary search with 96 entries
#
# This demonstrates the search TUI with real data
# Run this in your terminal (not via Claude Code)

set -e

cd "$(dirname "$0")"

echo "=== Diary Search Demo ==="
echo
echo "📊 Total entries: $(./diary testuser2 list | wc -l | tr -d ' ')"
echo
echo "🔍 Search examples to try:"
echo "  1. diary testuser2 search \"encryption\""
echo "  2. diary testuser2 search \"debugging\""
echo "  3. diary testuser2 search \"excited\""
echo "  4. diary testuser2 search \"API\""
echo
echo "Press any key to launch search for 'encryption'..."
read -n 1 -s
echo

./diary testuser2 search "encryption"

echo
echo "=== Demo complete ==="
