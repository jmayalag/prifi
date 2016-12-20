#variables
cothorityBranchRequired="master"
colors="true"
dbg_lvl=1
conf_file="config.toml"
group_file="group.toml"
prifi_file="prifi.toml"
bin_file="$GOPATH/src/github.com/lbarman/prifi_dev/sda/app/prifi.go"
colors="true"
port=8080
port_client=8090
configdir="config.demo"

#pretty print
shell="\e[35m[script]\e[97m"
errorMsg="\e[31m\e[1m[error]\e[97m\e[0m"
okMsg="\e[32m[ok]\e[97m"

print_usage() {
	echo
	echo -e "Usage: run-prifi.sh \e[33mrole/operation [params]\e[97m"
	echo -e "	\e[33mrole\e[97m: client, relay, trustee"
	echo -e "	\e[33moperation\e[97m: sockstest, all, deploy-all"
	echo -e "	\e[33mparams\e[97m for role \e[33mrelay\e[97m: [socks_server_port] (optional, numeric)"
	echo -e "	\e[33mparams\e[97m for role \e[33mtrustee\e[97m: id (required, numeric)"
	echo -e "	\e[33mparams\e[97m for role \e[33mclient\e[97m: id (required, numeric), [prifi_socks_server_port] (optional, numeric)"
	echo -e "	\e[33mparams\e[97m for operation \e[33mall\e[97m, \e[33mdeploy\e[97m: none"
	echo -e "	\e[33mparams\e[97m for operation \e[33msockstest\e[97m, \e[33mdeploy\e[97m: [socks_server_port] (optional, numeric), [prifi_socks_server_port] (optional, numeric)"
	echo
}

# tests if the cothority exists and is on the correct branch
test_cothority() {
	branchOk=$(cd $GOPATH/src/github.com/dedis/cothority; git status | grep "On branch $cothorityBranchRequired" | wc -l)

	if [ $branchOk -ne 1 ]; then
		echo -e "$errorMsg Make sure $GOPATH/src/github.com/dedis/cothority is a git repo, on branch $cothorityBranchRequired. Try to 'go get' the required packages !"
		exit 1
	fi
}

# test if $1 is a digit, if not, prints "argument $2 invalid" and exit.
test_digit() {
	case $1 in
		''|*[!0-9]*) 
			echo -e "$errorMsg parameter $2 need to be an integer."
			exit 1;;
		*) ;;
	esac
}

test_files() {

	if [ ! -f "$bin_file" ]; then
		echo -e "$errorMsg Runnable go file does not seems to exists: $bin_file"
		exit
	fi

	if [ ! -f "$configdir/$conf_file" ]; then
		echo -e "$errorMsg Cothority config file does not exist: $configdir/$conf_file"
		exit
	fi

	if [ ! -f "$configdir/$group_file" ]; then
		echo -e "$errorMsg Cothority group file does not exist: $configdir/$group_file"
		exit
	fi

	if [ ! -f "$configdir/$prifi_file" ]; then
		echo -e "$errorMsg PriFi config file does not exist: $configdir/$prifi_file"
		exit
	fi
}

#main switch, $1 is "operation" : "relay", "client", "trustee", "all"
case $1 in
	relay|Relay|RELAY)
	
		# the 2nd argument can replace the port number
		if [ "$#" -eq 2 ]; then
			test_digit $2 2
			port_client="$2"
		fi

		#specialize the config file, and test all files
		conf_file="relay/$conf_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$conf_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" relay
		;;

	trustee|Trustee|TRUSTEE)

		if [ "$#" -lt 2 ]; then
			echo -e "$errorMsg parameter 2 need to be the trustee id."
			exit 1
		fi
		test_digit $2 2

		#specialize the config file, and test all files
		conf_file="trustee$2/$conf_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$conf_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" trustee
		;;

	client|Client|CLIENT)
	
		if [ "$#" -lt 2 ]; then
			echo -e "$errorMsg parameter 2 need to be the client id."
			exit 1
		fi
		test_digit $2 2

		# the 3rd argument can replace the port number
		if [ "$#" -eq 3 ]; then
			test_digit $3 3
			port="$3"
		fi

		#specialize the config file, and test all files
		conf_file="client$2/$conf_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$conf_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" client
		;;

	sockstest|Sockstest|SOCKSTEST)
	
		# the 2rd argument can replace the port number
		if [ "$#" -gt 1 ]; then
			test_digit $2 2
			port="$2"
		fi

		# the 3rd argument can replace the port_client number
		if [ "$#" -eq 3 ]; then
			test_digit $3 3
			port_client="$3"
		fi

		#specialize the config file, and test all files
		conf_file="client0/$conf_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$conf_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" sockstest
		;;

	all|All|ALL|deploy|Deploy|DEPLOY|deploy-all|Deploy-All|DEPLOY-ALL)

		thisScript="$0"

		pkill prifi			

		socks=$(netstat -tunpl 2>/dev/null | grep $port_client | wc -l)
		
		if [ "$socks" -ne 1 ]; then
			echo -n "Socks proxy not running, starting it... "
			./run-socks-proxy.sh "$port_client" > socks.log 2>&1 &
			SOCKSPID=$!
			echo -e "$okMsg"
		fi

		echo -n "Starting relay...	"
		"$thisScript" relay > relay.log 2>&1 &
		RELAYPID=$!
		echo -e "$okMsg"

		sleep 3

		echo -n "Starting trustee 0...	"
		"$thisScript" trustee 0 > trustee0.log 2>&1 &
		TRUSTEE0PID=$!
		echo -e "$okMsg"

		sleep 3

		echo -n "Starting client 0...	"
		"$thisScript" client 0 8081 > client0.log 2>&1 &
		CLIENT0PID=$!
		echo -e "$okMsg"

		sleep 3

		echo -n "Starting client 1...	"
		"$thisScript" client 1 8082 > client1.log 2>&1 &
		CLIENT1PID=$!
		echo -e "$okMsg"

		sleep 3

		read -p "PriFi deployed. Press [enter] to kill all..." key

		kill -9 -$RELAYPID 2>/dev/null
		kill -9 -$TRUSTEE0PID 2>/dev/null
		kill -9 -$CLIENT0PID 2>/dev/null
		kill -9 -$CLIENT1PID 2>/dev/null
		kill -9 -$SOCKSPID 2>/dev/null
		pkill prifi		
		pkill run-server		
		;;

	clean|Clean|CLEAN)
		echo -n "Cleaning log files... "
		rm *.log 1>/dev/null 2>&1
		echo -e "$okMsg"
		;;

	*)
		print_usage
		;;
	
esac
