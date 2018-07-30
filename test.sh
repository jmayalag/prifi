#!/usr/bin/env bash


# ************************************
# PriFi all-in-one startup script
# ************************************
# author : Ludovic Barman
# email : ludovic.barman@gmail.com
# belongs to : the PriFi project
#           <github.com/dedis/prifi>
# ************************************

# variables that you might change often

dbg_lvl=3                       # 1=less verbose, 3=more verbose. goes up to 5, but then prints the SDA's message (network framework)
colors="true"                   # if  "false", the output of PriFi (and this script) will be in black-n-white

socksServer1Port=8080           # the port for the SOCKS-Server-1 (part of the PriFi client)
socksServer2Port=8090           # the port to attempt connect to (from the PriFi relay) for the SOCKS-Server-2
                                # notes : see <https://github.com/dedis/prifi/blob/master/README_architecture.md>

socks_test_n_clients=3          # number of clients to start in the "all-localhost" script

# default file names :

MAIN_SCRIPT="./prifi.sh"
THIS_SCRIPT="$0"

prifi_file="prifi.toml"                     # default name for the prifi config file (contains prifi-specific settings)
identity_file="identity.toml"               # default name for the identity file (contains public + private key)
group_file="group.toml"                     # default name for the group file (contains public keys + address of other nodes)

# location of the buildable (go build) prifi file :

bin_file="$GOPATH/src/github.com/dedis/prifi/sda/app/prifi.go"

# we have two "identities" directory. The second one is empty unless you generate your own keys with "gen-id"

configdir="config"
defaultIdentitiesDir="identities_default"   # in $configdir
realIdentitiesDir="identities_real"         # in $configdir

cothorityBranchRequired="master"            # the branch required for the cothority (SDA) framework

sleeptime_between_spawns=1                  # time in second between entities launch in "make it"

source "helpers.lib.sh"

# ------------------------
#     HELPER FUNCTIONS
# ------------------------

print_usage() {
    echo
    echo -e "PriFi, a tracking-resistant protocol for local-area anonymity"
    echo -e "** This is the testing module **. For advanced use only."
    echo
    echo -e "Usage: test.sh ${highlightOn}role/operation [params]${highlightOff}. Please check the .sh file for operations."
    echo 
}

run_relay() {
    #specialize the config file (we use the dummy folder, and maybe we replace with the real folder after)
    prifi_file2="$configdir/$prifi_file"
    identity_file2="$configdir/$defaultIdentitiesDir/relay/$identity_file"
    group_file2="$configdir/$defaultIdentitiesDir/relay/$group_file"
    test_files

    #run PriFi in relay mode
    echo "Config: $prifi_file2"
    DEBUG_COLOR="$colors" go run "$bin_file" --cothority_config "$identity_file2" --group "$group_file2" -d "$dbg_lvl" --prifi_config "$prifi_file2" --port "$socksServer1Port" --port_client "$socksServer2Port" relay
}

run_trustee() {
    trusteeId="$1"

    #specialize the config file (we use the dummy folder, and maybe we replace with the real folder after)
    prifi_file2="$configdir/$prifi_file"
    identity_file2="$configdir/$defaultIdentitiesDir/trustee$trusteeId/$identity_file"
    group_file2="$configdir/$defaultIdentitiesDir/trustee$trusteeId/$group_file"
    test_files

    #run PriFi in relay mode
    DEBUG_COLOR="$colors" go run "$bin_file" --cothority_config "$identity_file2" --group "$group_file2" -d "$dbg_lvl" --prifi_config "$prifi_file2" --port "$socksServer1Port" --port_client "$socksServer2Port" trustee
}

run_client() {
    clientId="$1"
    socksServer1Port="$2"

    #specialize the config file (we use the dummy folder, and maybe we replace with the real folder after)
    prifi_file2="$configdir/$prifi_file"
    identity_file2="$configdir/$defaultIdentitiesDir/client$clientId/$identity_file"
    group_file2="$configdir/$defaultIdentitiesDir/client$clientId/$group_file"
    test_files

    DEBUG_COLOR="$colors" go run "$bin_file" --cothority_config "$identity_file2" --group "$group_file2" -d "$dbg_lvl" --prifi_config "$prifi_file2" --port "$socksServer1Port" --port_client "$socksServer2Port" client
}

#test if all the files we need are there.
test_files() {

    if [ ! -f "$bin_file" ]; then
        echo -e "$errorMsg Runnable go file does not seems to exists: $bin_file"
        exit
    fi

    if [ ! -f "$identity_file2" ]; then
        echo -e "$errorMsg Cothority config file does not exist: $identity_file2"
        exit
    fi

    if [ ! -f "$group_file2" ]; then
        echo -e "$errorMsg Cothority group file does not exist: $group_file2"
        exit
    fi

    if [ ! -f "$prifi_file2" ]; then
        echo -e "$errorMsg PriFi config file does not exist: $prifi_file2"
        exit
    fi
}

run_integration_test_no_data() {
    # clean before start
    pkill prifi 2>/dev/null
    kill -TERM $(pidof "go run run-server.go") 2>/dev/null

    rm -f *.log

    # start all entities

    echo -n "Starting relay...                      "
    run_relay > relay.log 2>&1 &
    echo -e "$okMsg"
    sleep "$sleeptime_between_spawns"

    echo -n "Starting trustee 0...                  "
    run_trustee 0 > trustee0.log 2>&1 &
    echo -e "$okMsg"
    sleep "$sleeptime_between_spawns"

    echo -n "Starting client 0... (SOCKS on :8081)  "
    run_client 0 8081 > client0.log 2>&1 &
    echo -e "$okMsg"

    if [ "$socks_test_n_clients" -gt 1 ]; then
        sleep "$sleeptime_between_spawns"

        echo -n "Starting client 1... (SOCKS on :8082)  "
        run_client 1 8082 > client1.log 2>&1 &
        echo -e "$okMsg"
    fi

    if [ "$socks_test_n_clients" -gt 2 ]; then
        sleep "$sleeptime_between_spawns"

        echo -n "Starting client 2... (SOCKS on :8083)  "
        run_client 2 8083 > client2.log 2>&1 &
        echo -e "$okMsg"
    fi

    if [ "$socks_test_n_clients" -gt 3 ]; then
        echo -n "Max supported clients: 3, not booting any extra client."
    fi

    #let it boot
    waitTime=10
    echo "Waiting $waitTime seconds..."
    sleep "$waitTime"

    for repeat in 1 2 3
    do
        #reporting is every 5 second by default. if we wait 30, we should have 6 of those
        lines=$(cat relay.log | grep -E "([0-9\.]+) round/sec, ([0-9\.]+) kB/s up, ([0-9\.]+) kB/s down, ([0-9\.]+) kB/s down\(udp\)" | wc -l)

        echo "Number of reportings : $lines"

        if [ "$lines" -eq 0 ]; then
            echo "Waiting more time... $repeat/3"
            sleep 5
        else
            break
        fi

    done
    lines=$(cat relay.log | grep -E "([0-9\.]+) round/sec, ([0-9\.]+) kB/s up, ([0-9\.]+) kB/s down, ([0-9\.]+) kB/s down\(udp\)" | wc -l)

    if [ "$lines" -gt 0 ]; then
        echo "Test succeeded"
    else
        echo "Test failed"
        exit 1
    fi

    pkill prifi 2>/dev/null
    kill -TERM $(pidof "go run run-server.go")  2>/dev/null
}

do_socks_through_prifi() {
    port="$1"

    for repeat in 1 2 3
    do
        echo -en "Doing SOCKS HTTP request via :$port...   "
        curl google.com --socks5 127.0.0.1:$port --max-time 10 1>/dev/null 2>&1
        res=$?
        if [ "$res" -eq 0 ]; then
            echo -e "$okMsg"
            return 0
        else
            echo "Failed. Trying again... $repeat/3"
            sleep 5
        fi
    done

    echo "Test failed"
    exit 1
}

run_integration_test_ping() {
    pkill prifi 2>/dev/null
    pkill prifi-socks-server 2>/dev/null

    rm -f *.log

    #test if a socks proxy is already running (needed for relay), or start ours
    socks=$(netstat -tunpl 2>/dev/null | grep "$socksServer2Port" | wc -l)

    if [ "$socks" -ne 1 ]; then
        echo -n "Socks proxy not running, starting it..."
        cd socks && ./run-socks-proxy.sh "$socksServer2Port" > ../socks.log 2>&1 &
        SOCKSPID=$!
        echo -e "$okMsg"
    fi

    echo -n "Starting relay...                      "
    run_relay > relay.log 2>&1 &
    echo -e "$okMsg"

    sleep "$sleeptime_between_spawns"

    echo -n "Starting trustee 0...                  "
    run_trustee 0 > trustee0.log 2>&1 &
    echo -e "$okMsg"

    sleep "$sleeptime_between_spawns"

    echo -n "Starting client 0... (SOCKS on :8081)  "
    run_client 0 8081 > client0.log 2>&1 &
    echo -e "$okMsg"

    if [ "$socks_test_n_clients" -gt 1 ]; then
        sleep "$sleeptime_between_spawns"

        echo -n "Starting client 1... (SOCKS on :8082)  "
        run_client 1 8082 > client1.log 2>&1 &
        echo -e "$okMsg"
    fi

    if [ "$socks_test_n_clients" -gt 2 ]; then
        sleep "$sleeptime_between_spawns"

        echo -n "Starting client 2... (SOCKS on :8083)  "
        run_client 2 8083 > client2.log 2>&1 &
        echo -e "$okMsg"
    fi

    #let it boot
    waitTime=20
    echo "Waiting $waitTime seconds..."
    sleep "$waitTime"

    # first client
    do_socks_through_prifi 8081

    if [ "$socks_test_n_clients" -gt 1 ]; then
        # second client
        do_socks_through_prifi 8082
    fi

    if [ "$socks_test_n_clients" -gt 2 ]; then
        # third client
        do_socks_through_prifi 8083
    fi

    # cleaning everything

    pkill prifi 2>/dev/null
    pkill prifi-socks-server 2>/dev/null

    if [ "$res" -eq 0 ]; then
        echo "Test succeeded"
    else
        echo "Test failed"
        exit 1
    fi
}

# ------------------------
#     MAIN SWITCH
# ------------------------

# $1 is operation : "install", "relay", "client", "trustee", "sockstest", "all-localhost", "clean", "gen-id"
case $1 in

    sockstest|Sockstest|SOCKSTEST)

        #test for proper setup
        test_go
        test_cothority

        # the 2rd argument can replace the port number
        if [ "$#" -gt 1 ]; then
            test_digit "$2" 2
            socksServer1Port="$2"
        fi

        # the 3rd argument can replace the port_client number
        if [ "$#" -eq 3 ]; then
            test_digit "$3" 3
            socksServer2Port="$3"
        fi

        #specialize the config file, and test all files
        prifi_file2="$configdir/$prifi_file"
        identity_file2="$configdir/$defaultIdentitiesDir/relay/$identity_file"
        group_file2="$configdir/$defaultIdentitiesDir/relay/$group_file"
        test_files

        #run PriFi in relay mode
        DEBUG_COLOR="$colors" go run "$bin_file" --cothority_config "$identity_file2" --group "$group_file2" -d "$dbg_lvl" --prifi_config "$prifi_file2" --port "$socksServer1Port" --port_client "$socksServer2Port" sockstest
        ;;

    integration)

        echo "This test check that PriFi's clients, trustees and relay connect and start performing communication rounds with no real data."


        for f in "$configdir/"*-test.toml;
        do
            echo -e "Gonna test with ${highlightOn}$f${highlightOff}";
            prifi_file=$(basename "$f")
            run_integration_test_no_data
            sleep 5
        done

        echo -e "All tests passed."
        exit 0

        ;;

    integration2)

        echo "This test check that PriFi's clients, trustees and relay connect and start performing communication rounds, and that a Ping request can go through (back and forth)."

        if [ "$#" -eq 1 ]; then
            for f in "$configdir/"*-test.toml;
            do
                m=$(echo "$f" | grep "pcap" | wc -l) # do not use the test with replays pcap, it's incompatible with this
                if [ "$m" -eq 0 ]; then
                    echo -e "Gonna test with ${highlightOn}$f${highlightOff}";
                    prifi_file=$(basename "$f")
                    run_integration_test_ping
                fi
            done
        else
            f="$2"
            if [ ! -f "$f" ]; then
                echo -e "Cannot read file ${highlightOn}$f${highlightOff}"
                exit 1
            fi
            echo -e "Gonna test with ${highlightOn}$f${highlightOff}";
            prifi_file=$(basename "$f")
            run_integration_test_ping
        fi

        echo -e "All tests passed."
        exit 0
        
        ;;

    ping-through-prifi)

        #create a file ~/curl_format.cnf with this content
        #
        #    time_namelookup:  %{time_namelookup}\n
        #       time_connect:  %{time_connect}\n
        #    time_appconnect:  %{time_appconnect}\n
        #   time_pretransfer:  %{time_pretransfer}\n
        #      time_redirect:  %{time_redirect}\n
        # time_starttransfer:  %{time_starttransfer}\n
        #                    ----------\n
        #         time_total:  %{time_total}\n

        echo -n "Performing CURL through SOCKS:8081 to google.com, measuring latency..."

        for repeat in {1..10}
        do
            #curl -w "@curl_format.cnf" --socks5 127.0.0.1:8081 --max-time 10 -o /dev/null -s "http://google.com/"
            curl -w "@curl_format.cnf" --socks5 127.0.0.1:8081 --max-time 10 -o /dev/null -s "http://google.com/" > curl_ping_$repeat.txt
        done

        echo -e "$okMsg"
        ;;

    *)
        test_go
        test_cothority
        print_usage
        ;;
esac
