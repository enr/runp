
@echo OFF

:loop
echo infinite %DATE% %TIME%
PING -n 2 127.0.0.1>nul

goto loop
