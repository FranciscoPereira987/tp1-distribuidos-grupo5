#!/usr/bin/env bash
WORKERS_HEARTBEATER=${WORKERS_HEARTBEATER:-3}
WORKERS_QUERY1=${WORKERS_QUERY1:-1}
WORKERS_QUERY2=${WORKERS_QUERY2:-1}
WORKERS_QUERY3=${WORKERS_QUERY3:-1}
WORKERS_QUERY4=${WORKERS_QUERY4:-1}


echo '
#Port to be used by the service
net_port: 10000

#Port used by the beater protocol
heartbeat_port: 9000

#All peers, including self
peers:
'
for ((n = 1; n <= WORKERS_HEARTBEATER; n++))
do
echo "
  - peer_name: $n
    net_name: "peer$n"
"
done
echo '
#All containers to be monitored
containers:
'

for ((n = 1; n <= WORKERS_QUERY1; n++))
do
echo "- peer_name: demux_filter$n"
done

for ((n = 1; n <= WORKERS_QUERY2; n++))
do
echo "- peer_name: distance_filter$n" 
done

for ((n = 1; n <= WORKERS_QUERY3; n++))
do
echo "- peer_name: fastest_filter$n"
done

for ((n = 1; n <= WORKERS_QUERY4; n++))
do
echo "- peer_name: avg_filter$n"
done
  