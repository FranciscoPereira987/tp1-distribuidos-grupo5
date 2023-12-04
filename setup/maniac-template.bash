#!/usr/bin/env bash

interval=${INTERVAL:-%I}
victims=${VICTIMS:-%V}
peer_prefix=${PEER_PREFIX:-peer}
template='{{if not (eq .Names "rabbitmq" "input" "output")}}{{.Names}}{{end}}'

for ((;;)) do
    sleep $interval
    mapfile -t dying < <(docker ps --format "$template")
    # remove empty values
    dying=(${dying[@]})
    dying=(${dying[@]})
    for ((i=0; i<${#dying[@]}; i++)) 
    do
        if [[ ${dying[i]} =~ "client_" ]]
        then
            unset dying[i]
        fi
    done
    echo $dying
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
    unset survivor

    docker kill "${dying[@]}"
done
