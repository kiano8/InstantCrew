@echo off
echo Stopping PostgreSQL...
"C:\Users\yuri_\pgsql\bin\pg_ctl.exe" -D "C:\Users\yuri_\pgsql\data" stop
if %ERRORLEVEL% equ 0 (
    echo PostgreSQL stopped successfully.
) else (
    echo Failed to stop PostgreSQL or it is not running.
)
