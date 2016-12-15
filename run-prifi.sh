cothorityBranchRequired="master"

errorMsg="\e[31m\e[1m[error]\e[97m\e[0m"
okMsg="\e[32m[ok]\e[97m"

# tests if the cothority exists and is on the correct branch
test_cothority() {
	branchOk=$(cd $GOPATH/src/github.com/dedis/cothority; git status | grep "On branch $cothorityBranchRequired" | wc -l)

	if [ $branchOk -ne 1 ]; then
		echo -e "$errorMsg Make sure $GOPATH/src/github.com/dedis/cothority is a git repo, on branch master"
		exit 1
	fi
}

print_usage() {
	echo
	echo -e "Usage: run-prifi-standalone.sh \e[33mrole id\e[97m"
	echo -e "	\e[33mrole\e[97m: client, relay or trustee"
	echo -e "	\e[33mid\e[97m: integer (only for client or trustee roles)"
	echo
}


test_digit() {
	case $1 in
		''|*[!0-9]*) 
			echo -e "$errorMsg parameter 2 need to be an integer."
			print_usage;
			exit ;;
		*) ;;
	esac
}

# Argument validation

if [ "$#" -eq 1 ] && [ ! "$1" = "relay" ]; then
	echo -e "$errorMsg could not understand the parameters."
	print_usage
	exit
elif [ "$#" -eq 2 ]; then
	case "$1" in
		client|trustee) test_digit "$2" ;;
		*) 
			echo -e "$errorMsg could not understand the parameters."
			print_usage; 
			exit ;;
	esac
elif [ "$#" -gt 2 ]; then
	echo -e "$errorMsg could not understand the parameters."
	print_usage
	exit
elif [ "$#" -eq 0 ]; then
	#this time no error, as it is likely to be the first run by the user
	print_usage
	exit
fi


echo -e "Running command \e[1m./cd /sda/prifi_run/; ./run.sh $1 $2\e[0m"
cd ./sda/prifi_run/; ./run.sh $1  $2
