@echo off
REM Windows Local Test Script for OS Scanner

echo [TEST] Fast Mode (TTL Engine) - Should work (Accuracy ~80%)
..\..\neoAgent.exe scan os -t 127.0.0.1 --mode fast

echo.
echo [TEST] Deep Mode (Nmap Stack Engine) - Should FAIL gracefully on Windows (Not Supported)
..\..\neoAgent.exe scan os -t 127.0.0.1 --mode deep

echo.
echo [TEST] Auto Mode - Should fallback to TTL on Windows
..\..\neoAgent.exe scan os -t 127.0.0.1 --mode auto
