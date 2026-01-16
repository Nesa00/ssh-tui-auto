@echo off
title COMPILER AND RUNNER
color 70
:loop
certutil -hashfile "main.go" MD5
echo Compiling...
go build main.go
echo Completed
echo Run code
main.exe
echo CODE FINISHED RUNNING
pause
goto :loop