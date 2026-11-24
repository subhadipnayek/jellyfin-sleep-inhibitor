@echo off
:: Batch file to stop and remove Jellyfin Sleep Inhibitor Service
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo [ERROR] This script must be run as Administrator!
    echo Right-click this file and select "Run as administrator".
    pause
    exit /b 1
)

cd /d "%~dp0"
echo Stopping and uninstalling Jellyfin Sleep Inhibitor Service...
"%~dp0jellyfin-sleep-inhibitor.exe" stop
"%~dp0jellyfin-sleep-inhibitor.exe" uninstall
pause
