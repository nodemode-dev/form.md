@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"

set "BUNDLES=msi,nsis"
set "EXTRA="
set "SERVER_ONLY=0"
:args
if "%~1"=="" goto checked_args
if /i "%~1"=="--server-only" set "SERVER_ONLY=1" & shift & goto args
if /i "%~1"=="--debug" set "EXTRA=!EXTRA! --debug" & shift & goto args
if /i "%~1"=="--bundles" set "BUNDLES=%~2" & shift & shift & goto args
if /i "%~1"=="--config" set "EXTRA=!EXTRA! --config %2" & shift & shift & goto args
if /i "%~1"=="--help" goto usage
if /i "%~1"=="-h" goto usage
echo [build_app] unknown option: %~1
goto usage_fail

:checked_args
where go >nul 2>nul || (echo [build_app] Go 1.26 or newer is required: https://go.dev/dl/ & exit /b 1)
where node >nul 2>nul || (echo [build_app] Node 22 or newer is required: https://nodejs.org/ & exit /b 1)
if not defined GOTOOLCHAIN set "GOTOOLCHAIN=auto"
for /f "delims=" %%v in ('node -p "require('./web/package.json').version"') do set "APPV=%%v"
set "VERSION=v%APPV%"
echo [build_app] form.md %VERSION%

if not exist web\node_modules (
  echo [build_app] installing the web client's packages
  pushd web & call npm ci --no-audit --no-fund || (popd & exit /b 1)
  popd
)
echo [build_app] building the web client
pushd web & call npm run build || (popd & exit /b 1)
popd

echo [build_app] writing NOTICES.md
go run ./tools/notices || exit /b 1

if not exist bin mkdir bin
echo [build_app] compiling bin\form-md.exe
set "CGO_ENABLED=0"
go build -trimpath -ldflags "-s -w -X github.com/nodemode-ai/form.md/internal/app.Version=%VERSION%" -o bin\form-md.exe ./cmd/form-md || exit /b 1
if "%SERVER_ONLY%"=="1" (echo [build_app] done: bin\form-md.exe & exit /b 0)

where cargo >nul 2>nul || (echo [build_app] Rust is needed for the desktop app: install it from https://rustup.rs and run build_app.bat again & exit /b 1)
for /f "tokens=2" %%t in ('rustc -vV ^| findstr /b host:') do set "TRIPLE=%%t"
if not exist src-tauri\binaries mkdir src-tauri\binaries
del /q src-tauri\binaries\form-md-server-* 2>nul
copy /y bin\form-md.exe "src-tauri\binaries\form-md-server-%TRIPLE%.exe" >nul || exit /b 1
echo [build_app] server inside the app: src-tauri\binaries\form-md-server-%TRIPLE%.exe

findstr /c:"\"version\": \"%APPV%\"" src-tauri\tauri.conf.json >nul || (echo [build_app] src-tauri\tauri.conf.json does not carry version %APPV% & exit /b 1)
echo [build_app] bundling %BUNDLES%
pushd src-tauri
call ..\web\node_modules\.bin\tauri.cmd build --bundles %BUNDLES% %EXTRA%
set "STATUS=%ERRORLEVEL%"
popd
if not "%STATUS%"=="0" (echo [build_app] the desktop build failed; the log above shows the first error & exit /b %STATUS%)
echo [build_app] done. The installers are in src-tauri\target\release\bundle (msi and nsis)
echo [build_app] your pages live in %USERPROFILE%\form.md
exit /b 0

:usage
echo usage: build_app.bat [--server-only] [--debug] [--bundles msi,nsis] [--config JSON]
echo Builds the form.md desktop app for Windows with the server inside it.
exit /b 0
:usage_fail
echo usage: build_app.bat [--server-only] [--debug] [--bundles msi,nsis] [--config JSON]
exit /b 2
