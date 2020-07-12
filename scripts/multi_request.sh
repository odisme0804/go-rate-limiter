#!/bin/sh  
if [[ $# -ne 1 ]] ; then
    echo 'please pass a number to indicate the number of request'
    exit 1
fi

for i in $(seq 1 $1)
do  
    curl -i -X GET localhost:8080/request &
done

sleep 1
