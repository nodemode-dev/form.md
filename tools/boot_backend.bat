@echo off
cd /d "%~dp0\.."
if "%FORMMD_ADDR%"=="" set FORMMD_ADDR=127.0.0.1:8750
if "%FORMMD_DATA%"=="" set FORMMD_DATA=.\data
go run ./cmd/form-md serve
