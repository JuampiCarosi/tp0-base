#!/bin/bash

result=$(docker run --network=tp0_testing_net alpine  /bin/sh -c 'echo "test" | nc server 12345')

if [ $? -ne 0 -o "$result" != "test" ]; then
    echo "action: test_echo_server | result: fail"
    exit 1
fi

echo "action: test_echo_server | result: success"
