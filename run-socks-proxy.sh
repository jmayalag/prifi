colors="true"
dbg_lvl=3

cd socks;
DEBUG_COLOR=$colors go run run-server.go -debug="$dbg_lvl" -port="123456"
