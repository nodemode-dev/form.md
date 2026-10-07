@echo off
cd /d "%~dp0"
call tools\oc_install.bat || exit /b 1
call web\oc_install.bat
