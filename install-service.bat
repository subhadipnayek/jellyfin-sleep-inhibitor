@echo off
:: Batch file to install and start Jellyfin Sleep Inhibitor as an Administrator
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo [ERROR] This script must be run as Administrator!
    echo Right-click this file and select "Run as administrator".
    pause
    exit /b 1
)

cd /d "%~dp0"

if not exist config.json (
    if exist config.json.example (
        echo [INFO] config.json not found. Creating from config.json.example...
        copy config.json.example config.json
        echo [IMPORTANT] Please edit config.json and enter your Jellyfin API key before starting!
        pause
    )
)

echo Installing Jellyfin Sleep Inhibitor Service...
"%~dp0jellyfin-sleep-inhibitor.exe" install
if %errorLevel% equ 0 (
    echo Starting service...
    "%~dp0jellyfin-sleep-inhibitor.exe" start
    echo.
    echo Checking status...
    "%~dp0jellyfin-sleep-inhibitor.exe" status
)

pause
