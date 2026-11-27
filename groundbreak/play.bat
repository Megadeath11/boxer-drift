@echo off
rem Groundbreak launcher: gets the latest version, then starts the game.
cd /d "%~dp0"
title Groundbreak
echo Checking for updates...
git pull --ff-only
if errorlevel 1 (
  echo Could not update right now - starting the version you already have.
)
echo Preparing...
call npm install --no-audit --no-fund --loglevel=error
if errorlevel 1 (
  echo.
  echo npm failed. Is Node.js installed? Get the LTS version from https://nodejs.org
  pause
  exit /b 1
)
call npm start
