go clean -modcache
go mod tidy
go build -o geth.exe ./cmd/geth
pause