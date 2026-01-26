#!/bin/bash
# Linux Remote Test Script for OS Scanner
# Usage: ./test_os_scan_linux.sh [TARGET_IP]
# Default Target: 8.153.195.184

TARGET=${1:-"8.153.195.184"}

echo "---------------------------------------------------"
echo "Target: $TARGET"
echo "---------------------------------------------------"

echo "[TEST] Deep Mode (Nmap Stack Engine)"
echo "Expected: Precise OS info (Source: NmapStack) OR Error if ports closed"
./neoAgent scan os -t $TARGET --mode deep
echo "---------------------------------------------------"

echo "[TEST] Fast Mode (TTL Engine)"
echo "Expected: Approximate OS info (Source: TTL)"
./neoAgent scan os -t $TARGET --mode fast
echo "---------------------------------------------------"

echo "[TEST] Auto Mode"
echo "Expected: Best available result (Should be NmapStack if available)"
./neoAgent scan os -t $TARGET --mode auto
echo "---------------------------------------------------"
