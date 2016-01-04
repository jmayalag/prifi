@echo off
cd ..\\
go run main.go -relay -t1host=localhost:9000 -t2host=localhost:9000 -t3host=localhost:9000 -t4host=localhost:9000 -t5host=localhost:9000 -reportlimit=100 -logtype=netlogger
pause