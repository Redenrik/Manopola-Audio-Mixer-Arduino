@echo off
setlocal
cd /d "%~dp0\..\.."
go build -o dist\mama-quickstart\mama-ui.exe .\mama\cmd\mama-ui || exit /b 1
"%~dp0mama-ui.exe" %*
