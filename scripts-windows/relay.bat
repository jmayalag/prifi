@echo off
cd ..\\
go run main.go -node=prifi-relay -t1host=localhost:9000 -t2host=localhost:9000 -reportlimit=100
pause