@echo off
if not exist .\build\win-ci-compile.bat (
   echo This script must be run from the root of the repository.
   exit /b
)
if not defined GOPATH (
   echo GOPATH is not set.
   exit /b
)

set GOPATH=%GOPATH%;%cd%\Godeps\_workspace
set GOBIN=%cd%\build\bin

rem set gitCommit when running from a Git checkout.
set goLinkFlags=""
if exist ".git\HEAD" (
   where /q git
   if not errorlevel 1 (
      for /f %%h in ('git rev-parse HEAD') do (
          set goLinkFlags="-X main.gitCommit=%%h"
      )
   )
)

@echo on
go install -v -ldflags %goLinkFlags% ./...
