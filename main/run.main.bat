@echo off
title COMPILER AND RUNNER
rem color 70
:loop
certutil -hashfile "main.go" MD5
echo RUNNING
main.exe
echo CODE FINISHED RUNNING
pause
goto :loop