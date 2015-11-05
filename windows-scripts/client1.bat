@echo off
cd ..\\dissent\\
go run main.go config.go client.go relay.go relaySocks.go trusteeServer.go -client=1 -socks=false
pause