@echo off
echo Starting PhotoBridge...
echo.

cd /d "%~dp0backend"

REM Set default environment variables if not set
if "%ADMIN_USERNAME%"=="" set ADMIN_USERNAME=admin
if "%ADMIN_PASSWORD%"=="" set ADMIN_PASSWORD=admin123
if "%API_KEY%"=="" set API_KEY=photobridge-api-key
if "%JWT_SECRET%"=="" set JWT_SECRET=photobridge-jwt-secret
if "%PORT%"=="" set PORT=8080

echo Admin Username: %ADMIN_USERNAME%
echo Admin Password: %ADMIN_PASSWORD%
echo API Key: %API_KEY%
echo Server Port: %PORT%
echo.

go run main.go
