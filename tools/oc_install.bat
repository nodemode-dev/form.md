@echo off
cd /d "%~dp0\.."
where go >nul 2>nul || (echo Go 1.26 or newer is required: https://go.dev/dl/ & exit /b 1)
go mod download
go version
