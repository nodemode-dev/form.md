@echo off
cd /d "%~dp0"
if exist package-lock.json (
  call npm ci
) else (
  call npm install
)
