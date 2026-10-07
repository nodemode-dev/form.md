@echo off
cd /d "%~dp0"
start "form-md backend" cmd /k "tools\boot_backend.bat"
start "form-md frontend" cmd /k "web\boot_frontend.bat"
echo form.md dev: open http://127.0.0.1:5750 (API on http://127.0.0.1:8750)
