@echo off
title RUNNER WITH FLAGS
:loop                                                                
main.exe -a server -p 14022 -un root -pw secretpassword -loglevel 3 -showtime true -logtofile true -noyaml true                                                                   
echo CODE FINISHED RUNNING
pause
goto :loop