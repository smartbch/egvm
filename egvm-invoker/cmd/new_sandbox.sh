#!/bin/bash

#echo "$1"
exec ./nsjail -Me --disable_clone_newns --max_cpus=1 --time_limit=100000000  --cgroup_mem_max=536870912 --execute_fd -- "./egvmscript"