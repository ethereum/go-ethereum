@echo off
if not exist .\build\win-ci-test.bat (
   echo This script must be run from the root of the repository.
   exit /b
)
if not defined GOPATH (
   echo GOPATH is not set.
   exit /b
)

set GOPATH=%GOPATH%;%cd%\Godeps\_workspace
set GOBIN=%cd%\build\bin

@echo on
go test ./...
