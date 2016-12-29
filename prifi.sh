#!/usr/bin/env bash

#variables
cothorityBranchRequired="test_ism_2_699"
colors="true"
dbg_lvl=3
identity_file="identity.toml"
group_file="group.toml"
prifi_file="prifi.toml"
bin_file="$GOPATH/src/github.com/lbarman/prifi/sda/app/prifi.go"
colors="true"
port=8080
port_client=8090
configdir="config.localhost"
sleeptime_between_spawns=1
all_localhost_number_of_clients=2

#pretty print
shell="\e[35m[script]\e[97m"
errorMsg="\e[31m\e[1m[error]\e[97m\e[0m"
okMsg="\e[32m[ok]\e[97m"

print_usage() {
	echo
	echo -e "PriFi, a tracking-resistant protocol for local-area anonymity"
	echo
	echo -e "Usage: run-prifi.sh \e[33mrole/operation [params]\e[97m"
	echo -e "	\e[33mrole\e[97m: client, relay, trustee"
	echo -e "	\e[33moperation\e[97m: install, sockstest, all-localhost"
	echo -e "	\e[33mparams\e[97m for role \e[33mrelay\e[97m: [socks_server_port] (optional, numeric)"
	echo -e "	\e[33mparams\e[97m for role \e[33mtrustee\e[97m: id (required, numeric)"
	echo -e "	\e[33mparams\e[97m for role \e[33mclient\e[97m: id (required, numeric), [prifi_socks_server_port] (optional, numeric)"
	echo -e "	\e[33mparams\e[97m for operation \e[33minstall\e[97m: none"
	echo -e "	\e[33mparams\e[97m for operation \e[33mall-localhost\e[97m: none"
	echo -e "	\e[33mparams\e[97m for operation \e[33msockstest\e[97m: [socks_server_port] (optional, numeric), [prifi_socks_server_port] (optional, numeric)"
	echo

	echo -e "Man-page:"
	echo -e "	\e[33minstall\e[97m: get the dependencies, and tests the setup"
	echo -e "	\e[33mrelay\e[97m: starts a PriFi relay"
	echo -e "	\e[33mtrustee\e[97m: starts a PriFi trustee, using the config file trustee\e[33mid\e[97m"
	echo -e "	\e[33mclient\e[97m: starts a PriFi client, using the config file client\e[33mid\e[97m"
	echo -e "	\e[33mall-localhost\e[97m: starts a Prifi relay, a trustee, three clients all on localhost"
	echo -e "	\e[33msockstest\e[97m: starts the PriFi and non-PriFi SOCKS tunnel, without PriFi anonymization"
	echo -e "	Lost ? read https://github.com/lbarman/prifi/README.md"
}

#tests if GOPATH is set and exists
test_go(){
	if [ -z "$GOPATH"  ]; then
		echo -e "$errorMsg GOPATH is unset ! make sure you installed the Go language."
		exit 1
	fi
	if [ ! -d "$GOPATH"  ]; then
		echo -e "$errorMsg GOPATH ($GOPATH) is not a folder ! make sure you installed the Go language correctly."
		exit 1
	fi
}

# tests if the cothority exists and is on the correct branch
test_cothority() {
	branchOk=$(cd $GOPATH/src/github.com/dedis/cothority; git status | grep "On branch $cothorityBranchRequired" | wc -l)

	if [ $branchOk -ne 1 ]; then
		echo -e "$errorMsg Make sure \"$GOPATH/src/github.com/dedis/cothority\" is a git repo, on branch \"$cothorityBranchRequired\". Try running \"./prifi.sh install\""
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

	if [ ! -f "$configdir/$identity_file" ]; then
		echo -e "$errorMsg Cothority config file does not exist: $configdir/$identity_file"
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

#main switch, $1 is operation : "install", "relay", "client", "trustee", "sockstest", "all-localhost", "clean"
case $1 in

	install|Install|INSTALL)

		echo -n "Testing for GOPATH... "
		test_go
		echo -e "$okMsg"

		echo -n "Getting all go packages... "
		cd sda/app; go get ./... 1>/dev/null 2>&1
		cd ../..
		echo -e "$okMsg"

		echo -n "Switching cothority branch... "
		cd $GOPATH/src/github.com/dedis/cothority; git checkout "$cothorityBranchRequired" 1>/dev/null 2>&1
		echo -e "$okMsg"

		echo -n "Re-getting all go packages (since we switched branch)... "
		cd sda/app; go get ./... 1>/dev/null 2>&1
		cd ../..
		cd $GOPATH/src/github.com/dedis/cothority; go get ./... 1>/dev/null 2>&1
		echo -e "$okMsg"

		echo -n "Testing cothority branch... "
		test_cothority 
		echo -e "$okMsg"

		;;
	

	relay|Relay|RELAY)

		#test for proper setup
		test_go
		test_cothority
	
		# the 2nd argument can replace the port number
		if [ "$#" -eq 2 ]; then
			test_digit $2 2
			port_client="$2"
		fi

		#specialize the config file, and test all files
		identity_file="relay/$identity_file"
		group_file="relay/$group_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$identity_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" relay
		;;

	trustee|Trustee|TRUSTEE)

		#test for proper setup
		test_go
		test_cothority

		if [ "$#" -lt 2 ]; then
			echo -e "$errorMsg parameter 2 need to be the trustee id."
			exit 1
		fi
		test_digit $2 2

		#specialize the config file, and test all files
		identity_file="trustee$2/$identity_file"
		group_file="relay/$group_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$identity_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" trustee
		;;

	client|Client|CLIENT)

		#test for proper setup
		test_go
		test_cothority
	
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
		identity_file="client$2/$identity_file"
		group_file="relay/$group_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$identity_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" client
		;;

	sockstest|Sockstest|SOCKSTEST)

		#test for proper setup
		test_go
		test_cothority
	
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
		identity_file="client0/$identity_file"
		group_file="client0/$group_file"
		test_files

		#run PriFi in relay mode
		DEBUG_COLOR=$colors go run $bin_file --cothority_config "$configdir/$identity_file" --group "$configdir/$group_file" -d "$dbg_lvl" --prifi_config "$configdir/$prifi_file" --port "$port" --port_client "$port_client" sockstest
		;;

	localhost|Localhost|LOCALHOST|all-localhost|All-Localhost|ALL-LOCALHOST)

		#test for proper setup
		test_go
		test_cothority

		#test if a socks proxy is already running (needed for relay), or start ours
		socks=$(netstat -tunpl 2>/dev/null | grep $port_client | wc -l)
		
		if [ "$socks" -ne 1 ]; then
			echo -n "Socks proxy not running, starting it... "
			cd socks && ./run-socks-proxy.sh "$port_client" > ../socks.log 2>&1 &
			SOCKSPID=$!
			echo -e "$okMsg"
		fi

		thisScript="$0"	

		echo -n "Starting relay...			"
		"$thisScript" relay > relay.log 2>&1 &
		RELAYPID=$!
		echo -e "$okMsg"

		sleep $sleeptime_between_spawns

		echo -n "Starting trustee 0...			"
		"$thisScript" trustee 0 > trustee0.log 2>&1 &
		TRUSTEE0PID=$!
		echo -e "$okMsg"

		sleep $sleeptime_between_spawns

		echo -n "Starting client 0... (SOCKS on :8081)	"
		"$thisScript" client 0 8081 > client0.log 2>&1 &
		CLIENT0PID=$!
		echo -e "$okMsg"

        if [ "$all_localhost_number_of_clients" -gt 1 ]; then
            sleep $sleeptime_between_spawns

            echo -n "Starting client 1... (SOCKS on :8082)	"
            "$thisScript" client 1 8082 > client1.log 2>&1 &
            CLIENT1PID=$!
            echo -e "$okMsg"
		fi

        if [ "$all_localhost_number_of_clients" -gt 2 ]; then
            sleep $sleeptime_between_spawns

            echo -n "Starting client 2... (SOCKS on :8083)	"
            "$thisScript" client 2 8083 > client2.log 2>&1 &
            CLIENT1PID=$!
            echo -e "$okMsg"
		fi

		read -p "PriFi deployed. Press [enter] to kill all..." key

		kill -9 -$RELAYPID 2>/dev/null
		kill -9 -$TRUSTEE0PID 2>/dev/null
		kill -9 -$CLIENT0PID 2>/dev/null
		kill -9 -$CLIENT1PID 2>/dev/null
		kill -9 -$CLIENT2PID 2>/dev/null
		kill -9 -$SOCKSPID 2>/dev/null
		pkill prifi		
		pkill run-server # this is to kill the non-prifi SOCKS server. I am sure we can do better
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
