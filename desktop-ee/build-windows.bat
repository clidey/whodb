@echo off
REM Build script for Windows with optional code signing - both CE and EE editions

set EDITION=ce
if "%1"=="ee" set EDITION=ee

echo Building WhoDB Desktop Application for Windows - %EDITION% Edition...

REM Set variables based on edition
if "%EDITION%"=="ee" (
    set WORKSPACE=..\go.work.desktop-ee
    set BUILD_TAGS=-tags ee
    set OUTPUT_PREFIX=whodb-ee
    set BUILD_CMD=build:ee
) else (
    set WORKSPACE=..\go.work.desktop-ce
    set BUILD_TAGS=
    set OUTPUT_PREFIX=whodb-ce
    set BUILD_CMD=build:ce
)

cd ..\frontend
echo Building %EDITION% frontend...
call pnpm install
call pnpm run %BUILD_CMD%

cd ..\desktop-ee

REM Build for Windows AMD64
echo Building %EDITION% for Windows AMD64...
set GOWORK=%WORKSPACE%
wails build -clean -platform windows/amd64 ^
    %BUILD_TAGS% ^
    -nsis ^
    -windowsconsole=false ^
    -ldflags="-s -w -H windowsgui" ^
    -o %OUTPUT_PREFIX%-installer-amd64.exe

REM Build for Windows ARM64
echo Building %EDITION% for Windows ARM64...
wails build -clean -platform windows/arm64 ^
    %BUILD_TAGS% ^
    -nsis ^
    -windowsconsole=false ^
    -ldflags="-s -w -H windowsgui" ^
    -o %OUTPUT_PREFIX%-installer-arm64.exe

REM Optional: Sign the executable
REM Uncomment and configure the following lines if you have a code signing certificate:
REM echo Signing executables...
REM signtool sign /f "certificate.pfx" /p "password" /tr http://timestamp.digicert.com /td sha256 build\bin\%OUTPUT_PREFIX%-installer-amd64.exe
REM signtool sign /f "certificate.pfx" /p "password" /tr http://timestamp.digicert.com /td sha256 build\bin\%OUTPUT_PREFIX%-installer-arm64.exe

echo Build complete!
echo %EDITION% installers are in build\bin\
echo.
echo Usage: build-windows.bat [ce^|ee]
echo   ce - Build Community Edition (default)
echo   ee - Build Enterprise Edition