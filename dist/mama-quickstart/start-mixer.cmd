@echo off
setlocal
cd /d "%~dp0\..\.."
go build -o dist\mama-quickstart\mama.exe .\mama\cmd\mama || exit /b 1
"%~dp0mama.exe" %*
