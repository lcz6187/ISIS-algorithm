#!/bin/bash

trap "echo 'Terminating all processes...'; kill 0; exit" SIGINT SIGTERM

# Run each pipeline in the background
python3 -u gentx.py 10 | ./mp1_node node6 local_config_8.txt | python3 -u sanity.py &
python3 -u gentx.py 10 | ./mp1_node node7 local_config_8.txt | python3 -u sanity.py &
python3 -u gentx.py 10 | ./mp1_node node8 local_config_8.txt | python3 -u sanity.py &

wait
