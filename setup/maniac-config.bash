#!/usr/bin/env bash

WORKERS_HEARTBEATER=${WORKERS_HEARTBEATER:-3}
WORKERS_QUERY1=${WORKERS_QUERY1:-1}
WORKERS_QUERY2=${WORKERS_QUERY2:-1}
WORKERS_QUERY3=${WORKERS_QUERY3:-1}
WORKERS_QUERY4=${WORKERS_QUERY4:-1}
INTERVAL=${INTERVAL:-10s}
VICTIMS=${VICTIMS:-$((WORKERS_HEARTBEATER - 1))}

echo 'targets:'

for ((n=1; n<=WORKERS_HEARTBEATER; n++))
    do echo "  - peer$n"
done

for ((n=1; n<=WORKERS_QUERY1; n++))
    do echo "  - demux_filter$n"
done

for ((n=1; n<=WORKERS_QUERY2; n++))
    do echo "  - distance_filter$n"
done

for ((n=1; n<=WORKERS_QUERY3; n++))
    do echo "  - fastest_filter$n"
done

for ((n=1; n<=WORKERS_QUERY4; n++))
    do echo "  - avg_filter$n"
done

echo
echo interval: $INTERVAL
echo kill: $VICTIMS
