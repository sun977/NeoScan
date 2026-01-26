@echo off
REM Windows Local Test Script for OS Scanner
SET BIN_PATH=%~dp0..\..\neoAgent.exe

echo [TEST] Fast Mode (TTL Engine) - Should work (Accuracy ~80%%)
"%BIN_PATH%" scan os -t 127.0.0.1 --mode fast

echo.
echo [TEST] Deep Mode (Nmap Stack Engine) - Should FAIL gracefully on Windows (Not Supported)
"%BIN_PATH%" scan os -t 127.0.0.1 --mode deep

echo.
echo [TEST] Auto Mode - Should fallback to TTL on Windows
"%BIN_PATH%" scan os -t 127.0.0.1 --mode auto
