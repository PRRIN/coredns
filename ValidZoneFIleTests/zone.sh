#!/bin/bash

mkdir json -p

for file in ZoneFiles/*.txt; do
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
    sleep 1
    
    dig -p 1053 ${domain} SOA > /dev/null
    sleep 1
    
    mkdir "json/${filename}" -p
    
    mv zone.json "./json/${filename}/zone.json"
    mv request.json "./json/${filename}/request.json"
    echo "${file} complete"
    
    pkill coredns
    sleep 1
done
