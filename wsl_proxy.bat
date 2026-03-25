@echo off
title WSL Proxy
set /p ports="Portas (ex: 3000:3001 8000:8001): "
python "%~dp0wsl_proxy.py" %ports%
pause
