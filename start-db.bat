@echo off
echo Starting PostgreSQL...
"C:\Users\yuri_\pgsql\bin\pg_ctl.exe" -D "C:\Users\yuri_\pgsql\data" -l "C:\Users\yuri_\pgsql\server.log" start
if %ERRORLEVEL% equ 0 (
    echo PostgreSQL started successfully on port 5432.
) else (
    echo Failed to start PostgreSQL or it is already running.
)
