#!/bin/bash
# Test script for improved search UX
#
# This demonstrates three UX improvements:
# 1. Return to search after viewing entry (not exit to shell)
# 2. Interactive search input when no term provided
# 3. Press '/' to re-search with new term

set -e

cd "$(dirname "$0")"

echo "=== Search UX Improvements Test ==="
echo
echo "📋 Three improvements to test:"
echo
echo "1️⃣  After pressing Enter to view entry:"
echo "   → Detail view opens with Glamour-rendered markdown"
echo "   → Press 'q' to return to search results"
echo "   → Selection is preserved"
echo
echo "2️⃣  Start search without term:"
echo "   → Opens interactive search input"
echo "   → Type search term, press Enter"
echo "   → Shows results"
echo
echo "3️⃣  Press '/' to search for new term:"
echo "   → While viewing results, press '/'"
echo "   → Enter new search term"
echo "   → Updates results instantly"
echo
echo "Press any key to test improvement #1 (search with term)..."
read -n 1 -s
echo

echo "Running: ./diary testuser2 search \"encryption\""
echo "Try: Arrow keys to navigate, Enter to view, '/' for new search, 'q' to quit"
echo
./diary testuser2 search "encryption"

echo
echo "Press any key to test improvement #2 (no search term)..."
read -n 1 -s
echo

echo "Running: ./diary testuser2 search"
echo "Type a search term (e.g., 'debugging' or 'excited'), press Enter"
echo
./diary testuser2 search

echo
echo "=== All tests complete ==="
