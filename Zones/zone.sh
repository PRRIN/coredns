#!/bin/bash

mkdir json -p
mkdir json_filter -p

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
    sleep 0.2
    
    dig -p 1053 ${domain} SOA > /dev/null &
    sleep 0.2
    
    mkdir "json/${filename}" -p
    
    if ! grep -qE "CNAME|DNAME" "$file"; then
        mkdir "json_filter/${filename}" -p
        cp zone.json "./json_filter/${filename}/zone.json" 
        cp request.json "./json_filter/${filename}/request.json"
    fi

    mv zone.json "./json/${filename}/zone.json"
    mv request.json "./json/${filename}/request.json"
    echo "${file} complete"
    
    pkill coredns
    sleep 0.2
done
