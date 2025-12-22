#!/bin/bash

zone="Zones"
test_suite="$1"  

mkdir "${zone}"/json -p

for file in "${zone}"/"${test_suite}"/*.txt; do
    filename=$(basename "$file" .txt)
    
    domain=$(head -n 1 "$file" | awk '{print $1}')
    
    corefile=".:1053 {
    whoami
}

${domain}:1053 {
    file ${file}
}"

    echo "$corefile" > Corefile
    
    ./coredns > /dev/null &
    # Wait until coredns is running
    while ! pgrep -x "coredns" > /dev/null
    do
        sleep 0.1
    done

    # Repeat dig command until output is found
    until [ -f file.json -a -f request.json ]
    do
        dig @127.0.0.1 -p 1053 ${domain} ANY > /dev/null
        sleep 0.1
    done
    
    mkdir "${zone}/json/${filename}" -p
    
    mv file.json "${zone}/json/${filename}/file.json"
    mv request.json "${zone}/json/${filename}/request.json"
    # mv context.json "${zone}/json/${filename}/context.json"
    echo "${file} complete"
    
    pkill coredns

    # Make sure it's dead
    while pgrep -x "coredns" > /dev/null
    do
        sleep 0.1
    done
done
