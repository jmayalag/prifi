#!/usr/bin/env bash


# ************************************
# PriFi all-in-one startup script
# ************************************
# author : Ludovic Barman
# email : ludovic.barman@gmail.com
# belongs to : the PriFi project<github.com/dedis/prifi>
# ************************************

# variables that you might change often

dbg_lvl=3                                  # 1=less verbose, 3=more verbose. goes up to 5, but then prints the SDA's message (network framework)
colors="true"                              # if  "false", the output of PriFi (and this script) will be in black-n-white

# Experiment settings

SIMUL_FILE="prifi_simul.toml"
PLATFORM="deterlab"
EXEC_NAME="prifi_simul"
SIMUL_DIR="sda/simulation"
DETERLAB_USER="lbarman"
MPORT="10002"

TEMPLATE_FILE="sda/simulation/prifi_simul_template.toml"
CONFIG_FILE="sda/simulation/prifi_simul.toml"

SIMULATION_TIMEOUT="400"                   # note: in simul-vary-xxx, this is the timeout of *one* tick, no the whole loop

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

sleeptime_between_spawns=1                  # time in second between entities launch in all-localhost part

    
DETERLAB_PCAP_LOCATION='/users/lbarman/remote/pcap/'

source "helpers.lib.sh"

# ------------------------
#     HELPER FUNCTIONS
# ------------------------

print_usage() {
    echo
    echo -e "PriFi, a tracking-resistant protocol for local-area anonymity"
    echo -e "** This is the simulation interface that interacts with Deterlab **. For advanced use only."
    echo
    echo -e "Usage: simul.sh ${highlightOn}role/operation [params]${highlightOff}. Please check the .sh file for operations."
    echo 
}

setPCAPOnServer() {

    traffic=$1
    active_hosts=$2

    echo "Removing old symlinks"
    ssh $DETERLAB_USER@users.deterlab.net "rm -f ${DETERLAB_PCAP_LOCATION}client*.pcap"

    echo "Setting $active_hosts to replay $traffic"
    for (( i=0; i<$active_hosts; i++ ))
    do
        echo "Linking $traffic to client$i.pcap"
        ssh $DETERLAB_USER@users.deterlab.net "ln -s ${DETERLAB_PCAP_LOCATION}${traffic} ${DETERLAB_PCAP_LOCATION}client$i.pcap"
    done
}

simul-helper() {
    window=$1
    upCellSize=$2
    clients=$3
    repeat=$4
    traffic=$5

    NTRUSTEES=3
    NRELAY=1
    percentage_clients="5"

    active_hosts=`echo "scale=2; 0.5+$percentage_clients/100*$clients" | bc`
    active_hosts=`printf %.0f $active_hosts`
    hosts=$(($NTRUSTEES + $NRELAY + $clients))

    echo "Simulating for TRAFFIC $traffic, CLIENTS=$clients, ACTIVE_CLIENTS=$active_hosts, UCS $upCellSize, win $window REPEAT ${repeat}..."
    setPCAPOnServer "$traffic" "$active_hosts"
    $(cd ./sda/simulation && ./setparam.py "Hosts=$hosts" "RelayWindowSize=$window" "CellSizeUp=$upCellSize")
    timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${traffic}_${clients}_${active_hosts}_${repeat}.txt
}

# ------------------------
#     MAIN SWITCH
# ------------------------

# $1 is operation : "simul", "simul-ping", "simul-get-logs", "simul-clear-logs", "simul-vary-nclients", "simul-vary-nclients2", "simul-mcast-rules", etc
case $1 in

    simul|Simul|SIMUL)



        EXPERIMENT_ID_VALUE=$(LC_ALL=C cat /dev/urandom | LC_ALL=C tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
        dbg_lvl=1 # do not change this. Many other functions of this script call this script recursively. If this is >1, the log will blow up ;)

        rm -f last-simul.log

        echo -n "Building simulation... " | tee last-simul.log
        cd "$SIMUL_DIR"; go build -o "$EXEC_NAME" *.go | tee ../../last-simul.log
        echo -e "$okMsg" | tee ../../last-simul.log

        echo -en "Simulation ID is ${highlightOn}${EXPERIMENT_ID_VALUE}${highlightOff}, storing it in ${highlightOn}~/remote/.simID${highlightOff} on remote... " | tee ../../last-simul.log
        ssh $DETERLAB_USER@users.deterlab.net "echo ${EXPERIMENT_ID_VALUE} > ~/remote/.simID"  | tee ../../last-simul.log
        ssh $DETERLAB_USER@users.deterlab.net "rm -f ~/remote/.lastsimul"
        echo -e "$okMsg" | tee ../../last-simul.log

        echo -e "Starting simulation ${highlightOn}${SIMUL_FILE}${highlightOff} on ${highlightOn}${PLATFORM}${highlightOff}." | tee ../../last-simul.log
        DEBUG_LVL=$dbg_lvl DEBUG_COLOR=$colors ./"$EXEC_NAME" -platform "$PLATFORM" -mport "$MPORT" "$SIMUL_FILE" | tee ../../last-simul.log

        echo -n "Simulation done, cleaning up... " | tee ../../last-simul.log
        rm -f "$EXEC_NAME" | tee ../../last-simul.log
        echo -e "$okMsg" | tee ../../last-simul.log
        
        status=$(ssh $DETERLAB_USER@users.deterlab.net "cat ~/remote/.lastsimul")
        echo -e "Status is ${highlightOn}${status}${highlightOff}." | tee ../../last-simul.log

        ;;

    simul-p|simul-ping)

        #create a file ~/pings.sh with this content
        #  #!/bin/sh
        #  for ip in 10.0.1.1 10.1.0.1; do
        #      echo "Pinging $ip"
        #      ssh relay.LB-LLD.SAFER.isi.deterlab.net "ping $ip -w 10 -c 10 | grep rtt"
        #      echo -n ";"
        #  done
        # [EOF]

        echo -n "Mesuring latencies... "
        pings=$(ssh $DETERLAB_USER@users.deterlab.net "./pings.sh")
        echo -e "$okMsg"
        echo $pings | sed -e "s/10.0.1.1/client0/" | sed -e "s/10.1.0.1/trustee0/" | tr ';' '\n'
        ;;

    simul-gl|simul-get-logs)

        #create a file ~/makelogsrw.sh with this content
        #   #!/bin/sh
        #   ssh relay.LB-LLD.SAFER.isi.deterlab.net 'cd remote; sudo chmod ugo+rw -R .'
        # [EOF]

        expFolder="experiment_out"

        echo -e "${warningMsg} Note that this tool downloads every log on the server. If you forgot to clean them, it might concern serveral experiments."

        echo -n "Making logs R/W... " #this is needed since simul runs and writes log as root
        ssh $DETERLAB_USER@users.deterlab.net './makelogsrw.sh'
        echo -e "$okMsg"

        read -p "Which name do you want to give the data on the server ? " expName

        if [ -d "$expFolder/$expName" ]; then
            echo -e "${errorMsg} Directory ${highlightOn}$expFolder/$expName${highlightOff} already exists, exiting."
            exit 1
        fi

        echo -ne "Making folder ${highlightOn}$expFolder/$expName${highlightOff} "
        mkdir -p "$expFolder/$expName"
        echo -e "$okMsg"

        echo -ne "Fetching all experiments of the form ${highlightOn}output_*${highlightOff} "
        cd "$expFolder/$expName";
        out=$(scp -r $DETERLAB_USER@users.deterlab.net:~/remote/output_\* . )
        echo -e "$okMsg"

        echo -ne "Writing the download date... "
        date > "download_date"
        echo -e "$okMsg"

        echo -ne "Changing rights back to something normal... ${highlightOn}u+rwx,go-rwx${highlightOff} "
        chmod u+rwx -R .
        chmod go-rwx -R .
        echo -e "$okMsg"

        echo "Copied files are :"
        echo ""
        cd ..
        tree -a "$expName"

        ;;

    simul-cl|simul-clear-logs)

        #create a file ~/makelogsrw.sh with this content
        #   #!/bin/sh
        #   ssh relay.LB-LLD.SAFER.isi.deterlab.net 'cd remote; sudo chmod ugo+rw -R .'
        # [EOF]

        echo -e "${warningMsg} This tool *deletes* all experiment data on the remote server. Make sure you backuped what you need !"

        read -p "Would you like to continue and *delete* all logs [y/n] ? " ans

        if [ $ans = y -o $ans = Y -o $ans = yes -o $ans = Yes -o $ans = YES ]
        then

            echo -n "Making logs R/W... " #this is needed since simul runs and writes log as root
            ssh $DETERLAB_USER@users.deterlab.net './makelogsrw.sh'
            echo -e "$okMsg"

            echo -n "Deleting all remote logs... "
            ssh $DETERLAB_USER@users.deterlab.net 'cd remote; rm -rf output_*;'
            echo -e "$okMsg"

        else
            echo "Aborting without taking any action."
        fi

        ;;

    simul-mcast-rules|simul-mr)

        #create a file ~/mcast2.sh with this content
        # #!/bin/sh
        # iface=$(ip addr | sed -r ':a;N;$!ba;s/\n\s/ /g' | sed -r -n -e 's/^([0-9]+):\s(\w+).*(link\/(\w+))\s[a-f0-9:.]{,17}\sbrd\s[a-f0-9:.]{,17}\s*(inet\s([0-9]{1,3}(\.[0-9]{1,3}){3})).*/\2 \6 \4/p' -e 's/^([0-9]+):\s(\w+).*(link\/(\w+))\s[a-f0-9:.]{,17}\sbrd\s[a-f0-9:.]{,17}.*/\2 0.0.0.0 \4/p' | grep 10.0.1 | cut -d ' ' -f 1)
        # echo "Redirecting mcast traffic to $iface"
        # sudo route del -net 224.0.0.0/8
        # sudo route add -net 224.0.0.0/8 "$iface"
        # [EOF]

        #create a file ~/mcast.sh with this content
        # #!/bin/sh
        # echo "Connecting to relay"
        # ssh relay.LB-LLD.SAFER.isi.deterlab.net './mcast2.sh'
        # for i in 0 1 2 3 4; do
        #     echo "Connecting to client-$i"
        #     ssh client-$i.LB-LLD.SAFER.isi.deterlab.net './mcast2.sh'
        # done
        # [EOF]
        
        echo -n "Setting multicast to go through 10.0.1.0/8 network... "
        ssh $DETERLAB_USER@users.deterlab.net './mcast.sh'
        echo -e "$okMsg"
        ;;

    simul-vary-nclients)

        NTRUSTEES=3
        NRELAY=1

        "$THIS_SCRIPT" simul-cl

        for repeat in {1..4}
        do
            for i in 10 20 30 40 50 60 70 80 90
            do
                hosts=$(($NTRUSTEES + $NRELAY + $i))
                echo "Simulating for HOSTS=$hosts..."

                #fix the config
                rm -f "$CONFIG_FILE"
                sed "s/Hosts = x/Hosts = $hosts/g" "$TEMPLATE_FILE" > "$CONFIG_FILE"

                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${i}_${repeat}.txt
            done
        done

        ;;

    simul-vary-sleep)

        NTRUSTEES=3
        NRELAY=1

        "$THIS_SCRIPT" simul-cl

        for repeat in {1..10}
        do
            for i in {0..100..10}
            do
                echo "Simulating for Delay=$i..."

                #fix the config
                rm -f "$CONFIG_FILE"
                sed "s/OpenClosedSlotsMinDelayBetweenRequests = x/OpenClosedSlotsMinDelayBetweenRequests = $i/g" "$TEMPLATE_FILE" > "$CONFIG_FILE"

                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${i}_${repeat}.txt
            done
        done

        ;;

    simul-vary-window)

        "$THIS_SCRIPT" simul-cl

        for repeat in {1..3}
        do
            for window in {1..10}
            do
                echo "Simulating for WINDOW=$window..."

                #fix the config
                rm -f "$CONFIG_FILE"
                sed "s/RelayWindowSize = x/RelayWindowSize = $window/g" "$TEMPLATE_FILE" > "$CONFIG_FILE"

                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${window}_${repeat}.txt
            done
        done
        ;;

    simul-vary-upstream)

        "$THIS_SCRIPT" simul-cl

        for repeat in {1..10}
        do
            for upsize in 1000 2000 3000 4000 5000 6000 7000 8000 9000 10000
            do
                echo "Simulating for upsize=$upsize  (repeat $repeat)..."

                #fix the config
                rm -f "$CONFIG_FILE"
                sed "s/CellSizeUp = x/CellSizeUp = $upsize/g" "$TEMPLATE_FILE" > "$CONFIG_FILE"

                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${upsize}_${repeat}.txt
            done
        done
        ;;

    simul-vary-downstream)

        "$THIS_SCRIPT" simul-cl

        for repeat in {1..10}
        do
            for downsize in 17400 17500 17600 17800 17900 18000
            do
                echo "Simulating for downsize=$downsize  (repeat $repeat)..."

                #fix the config
                rm -f "$CONFIG_FILE"
                sed "s/CellSizeDown = x/CellSizeDown = $downsize/g" "$TEMPLATE_FILE" > "$CONFIG_FILE"

                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${downsize}_${repeat}.txt
            done
        done
        ;;

    simul-e|simul-edit)

        nano sda/simulation/prifi_simul.toml
        ;;



    simul-vary-dcnet)

        NTRUSTEES=3
        NRELAY=1

        "$THIS_SCRIPT" simul-cl

        for repeat in {1..5}
        do
            for i in {0..3}
            do
                dis="false"
                equiv="false"

                if [ $i == 1 ]; then
                    dis="true"
                fi
                if [ $i == 2 ]; then
                    equiv="true"
                fi
                if [ $i == 3 ]; then
                    dis="true"
                    equiv="true"
                fi

                echo "Simulating for DCNET disruption=$dis, equivocation=$equiv, repeat $repeat"


                #fix the config
                $(cd ./sda/simulation && ./setparam.py "DisruptionProtectionEnabled=$dis" "EquivocationProtectionEnabled=$equiv")

                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${i}_${repeat}.txt
            done
        done

        ;;

    simul-vary-workloads)

        NTRUSTEES=3
        NRELAY=1

        #"$THIS_SCRIPT" simul-cl

        for traffic in skype.pcap # hangouts.pcap others.pcap youtube.pcap
        do
            for repeat in {1..20}
            do
                for clients in 10 20 30 # 30 50 70 90
                do
                    for percentage_clients in 1 # 5
                    do
                        hosts=$(($NTRUSTEES + $NRELAY + $clients))
                        active_hosts=`echo "scale=2; 0.5+$percentage_clients/100*$hosts" | bc`
                        active_hosts=`printf %.0f $active_hosts`

                        echo "Simulating for TRAFFIC $traffic, CLIENTS=$clients, ACTIVE_CLIENTS=$active_hosts, REPEAT ${repeat}..."

                        setPCAPOnServer "$traffic" "$active_hosts"

                        #fix the config
                        rm -f "$CONFIG_FILE"
                        sed "s/Hosts = x/Hosts = $hosts/g" "$TEMPLATE_FILE" > "$CONFIG_FILE"

                        timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${traffic}_${clients}_${active_hosts}_${repeat}.txt
                    done
                done
            done

        done
        ;;

    simul-vary-workloads-deep)
    
        NTRUSTEES=3
        NRELAY=1

        #"$THIS_SCRIPT" simul-cl

        for repeat in {1..5}
        do
            for traffic in hangouts.pcap # skype.pcap others.pcap youtube.pcap
            do
                for clients in 60 70 80 90 #10 30 30 50 70 90
                do
                    for percentage_clients in 5
                    do
                        for window in 5 6 7 8 9 10
                        do
                            for upCellSize in 4000 5000 6000 7000 8000 9000 10000 11000 12000
                            do
                                hosts=$(($NTRUSTEES + $NRELAY + $clients))
                                active_hosts=`echo "scale=2; 0.5+$percentage_clients/100*$hosts" | bc`
                                active_hosts=`printf %.0f $active_hosts`

                                echo "Simulating for TRAFFIC $traffic, CLIENTS=$clients, ACTIVE_CLIENTS=$active_hosts, window=$window, UCS=$upCellSize, REPEAT ${repeat}..."

                                setPCAPOnServer "$traffic" "$active_hosts"

                                #fix the config
                                $(cd ./sda/simulation && ./setparam.py "Hosts=$hosts" "RelayWindowSize=$window" "CellSizeUp=$upCellSize")

                                timeout "$SIMULATION_TIMEOUT" "$THIS_SCRIPT" simul | tee experiment_${traffic}_${clients}_${active_hosts}_${upCellSize}_${window}_${repeat}.txt
                            done
                        done
                    done
                done
            done

        done
        ;;

    simul-skype)

        for repeat in {1..5}
        do
            simul-helper 5 8000 10 $repeat "skype.pcap"
            simul-helper 5 8000 20 $repeat "skype.pcap"
            simul-helper 5 8000 30 $repeat "skype.pcap"
            simul-helper 4 8000 40 $repeat "skype.pcap"
            simul-helper 3 7000 50 $repeat "skype.pcap"
            simul-helper 4 6000 60 $repeat "skype.pcap"
            simul-helper 3 5000 70 $repeat "skype.pcap"
            simul-helper 4 6000 80 $repeat "skype.pcap"
            simul-helper 5 7000 90 $repeat "skype.pcap"
        done
        ;;

    simul-others)

        for repeat in {1..5}
        do
            simul-helper 7 5000 10 $repeat "others.pcap"
            simul-helper 7 5000 20 $repeat "others.pcap"
            simul-helper 5 5000 30 $repeat "others.pcap"
            simul-helper 5 5000 40 $repeat "others.pcap"
            simul-helper 3 5000 50 $repeat "others.pcap"
            simul-helper 3 5000 60 $repeat "others.pcap"
            simul-helper 3 5000 70 $repeat "others.pcap"
            simul-helper 3 5000 80 $repeat "others.pcap"
            simul-helper 3 5000 90 $repeat "others.pcap"
        done
        ;;

    simul-hangout)

        for repeat in {1..5}
        do
            simul-helper 7 5000 10 $repeat "hangouts.pcap"
            simul-helper 7 5000 20 $repeat "hangouts.pcap"
            simul-helper 5 5000 30 $repeat "hangouts.pcap"
            simul-helper 5 5000 40 $repeat "hangouts.pcap"
            simul-helper 5 8000 50 $repeat "hangouts.pcap"
            simul-helper 5 8000 60 $repeat "hangouts.pcap"
            simul-helper 5 7000 70 $repeat "hangouts.pcap"
            simul-helper 5 7000 80 $repeat "hangouts.pcap"
            simul-helper 5 10000 90 $repeat "hangouts.pcap"
        done
        ;;

    simul-youtube)
        for repeat in {1..5}
        do
            simul-helper 7 5000 10 $repeat "youtube.pcap"
            simul-helper 5 5000 20 $repeat "youtube.pcap"
            simul-helper 5 5000 30 $repeat "youtube.pcap"
            simul-helper 5 5000 40 $repeat "youtube.pcap"
            simul-helper 3 5000 50 $repeat "youtube.pcap"
            simul-helper 3 5000 60 $repeat "youtube.pcap"
            simul-helper 3 5000 70 $repeat "youtube.pcap"
            simul-helper 3 5000 80 $repeat "youtube.pcap"
            simul-helper 3 5000 90 $repeat "youtube.pcap"
        done
        ;;

    *)
        test_go
        test_cothority
        print_usage
        ;;

esac
