#!/bin/bash
# Run diary and capture full panic output
./diary search testuser2 2>&1 | tee /tmp/diary_panic.log
