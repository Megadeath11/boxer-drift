@echo off
rem Builds a portable ElementSandbox.exe on Windows. Requires Go: https://go.dev/dl/
cd /d "%~dp0"
if not exist dist mkdir dist
set CGO_ENABLED=0
go build -trimpath -ldflags "-H=windowsgui -s -w" -o dist\ElementSandbox.exe .
if %errorlevel% neq 0 exit /b %errorlevel%
echo Built dist\ElementSandbox.exe
