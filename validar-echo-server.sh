#!/bin/bash

docker run --network=tp0_testing_net alpine  /bin/sh -c 'apk add netcat-openbsd > /dev/null && nc -z -v server 12345'  > /dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "action: test_echo_server | result: fail"
    exit 1
fi

echo "action: test_echo_server | result: success"
