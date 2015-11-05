@echo off
cd ..\\dissent\\
go run main.go config.go client.go relay.go relaySocks.go trusteeServer.go -client=0 -socks=false
pause