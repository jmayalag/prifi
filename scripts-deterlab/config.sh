programpath="/users/lbarman/dissent/"
program="prifi"
programFreeBSD="prifi-freebsd-amd64"
useUdp="false"
socks="false"
relayhostaddr="10.0.0.254:9876"

loglevel=5 #log everything
netLogStdOut="true" #also output log to STDOUT
logPath="/tmp/"
logtype="netlogger" #or "file"
loghost="192.168.253.1:10000"

logsinkname="sink.log"

nohupoutfolder="/tmp/"
nohupclientname="client"
nohuprelayname="relay"
nohupsinkname="sink"
nohuptrusteesrvname="trusteesrv"
nohupext=".nohup"

t1host="10.0.1.1:9000"
t2host="10.0.1.2:9000"
t3host="10.0.1.3:9000"
t4host="10.0.1.4:9000"
t5host="10.0.1.5:9000"

tXhostsString="-t1host=$t1host -t2host=$t2host -t3host=$t3host -t4host=$t4host -t5host=$t5host"

logParamsString="-loglvl=$loglevel -udp=$useUdp -logtostdout=$netLogStdOut -logpath=$logPath -logtype=$logtype -loghost=$loghost"