@echo off
cd ..\\
go run main.go -node=prifi-client-0 -socks=false -logtype=netlogger -latencytest=true
pause