@echo off
cd ..\\dissent\\
go run main.go config.go relay.go trusteeServer.go -client=2 -socks=false