#!/usr/bin/env bash

interval=${INTERVAL:-%I}
victims=${VICTIMS:-%V}
peer_prefix=${PEER_PREFIX:-peer}
template='{{if and (ne .Names "rabbitmq") (ne .Image "client")}}{{.Names}}{{end}}'

for ((;;)) do
    sleep $interval
    mapfile -t dying < <(docker ps --format "$template")
    # remove empty values
    dying=(${dying[@]})
    if [[ ${#dying[@]} -le $victims ]]
    then
        continue
    fi

    for index in $(shuf -n $((${#dying[@]} - victims)) -i0-$((${#dying[@]}-1)))
    do
        if [[ ${dying[index]} =~ $peer_prefix ]]
        then
            survivor=${dying[index]}
        fi
        last=${dying[index]}
        unset dying[index]
    done

    if [[ -z $survivor ]]
    then
        for ((i = 0;; i++)) do
            if [[ ${dying[i]} =~ $peer_prefix ]]
            then
                echo rescued ${dying[i]}
                dying[i]=$last
                break
            fi
        done
    fi
    unset survivor last

    docker kill "${dying[@]}"
done
