#!/bin/sh

DIR="$( cd "$( dirname "$SCRIPT" )" >/dev/null 2>&1 && pwd )"

TEST_SUITES=$DIR/suite
CARBON_RELAY_NG_BIN="${CARBON_RELAY_NG_BIN:-$DIR/../carbon-relay-ng}"
NETCAT_BIN="${NETCAT_BIN:-nc}"

TCP_INPUT_PORT="${TCP_INPUT_PORT:-2003}"
UDP_INPUT_PORT="${UDP_INPUT_PORT:-2003}"
TCP_RECEIVER_PORT="${TCP_RECEIVER_PORT:-9999}"

config_files=$(find $TEST_SUITE -name "*_config" | sort)
input_files=$(find $TEST_SUITE -name "*_input" | sort)

start_carbon_ng() {
	config_file=$1
	$CARBON_RELAY_NG_BIN $config_file &
	pid=$!
	sleep 1
	return $pid
}

start_receiver() {
	output_file=$1
	$NETCAT_BIN -kl $TCP_RECEIVER_PORT > $output_file &
	pid=$!
	sleep 1
	return $pid
}

stop_process() {
	pid=$1
	kill $pid
	wait $pid
}

inject_payload() {
	input_file=$1
	output_file=$2

	cat $input_file | $NETCAT_BIN -q0 localhost $TCP_INPUT_PORT > $output_file
}

run_test() {
	input_file=$1
	output_file=${input_file#_input#_output}
	expected_file=${input_file#_input#_expected}
	test_status=1

	pid=start_receiver $output_file

	inject_payload "$input_file" "$output_file"
	
	sleep 1
	stop_process $pid
	
	diff -s $expected_file $output_file > /dev/null
	if [ $? -eq 0 ]
	then
		test_status=0
		echo "Test $(basename $input_file) ... OK"
		rm $output_file
	else
		echo "Test $(basename $input_file) ... FAILED"
	fi
}

status=0
for config_file in "$config_files"
do
	pid=$(start_carbon_ng $config_file)
	for input_file in "$input_files"
	do
		status=run_test $input_file
		if [ -n status -eq 0 ]
		then
			break
		fi	
	done
	stop_process $pid
	if [ -n status -eq 0 ]
	then
		break
	fi	
done

exit $status
