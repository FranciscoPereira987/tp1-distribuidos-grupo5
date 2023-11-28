#!/usr/bin/env bash
WORKERS_HEARTBEATER=${WORKERS_HEARTBEATER:-3}
WORKERS_QUERY1=${WORKERS_QUERY1:-1}
WORKERS_QUERY2=${WORKERS_QUERY2:-1}
WORKERS_QUERY3=${WORKERS_QUERY3:-1}
WORKERS_QUERY4=${WORKERS_QUERY4:-1}

echo '
name: tp1
services:

  rabbitmq:
    container_name: rabbitmq
    image: rabbitmq:3.12-management
    ports:
      - 15672:15672
    networks:
      - testing_net
    healthcheck:
      test: rabbitmq-diagnostics check_port_connectivity
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 50s'

echo "
  inputBoundary:
    container_name: input
    image: input_boundary:latest
    entrypoint: /inputBoundary
    networks:
      - testing_net
    environment:
      - IN_WORKERS=$((WORKERS_QUERY1 + WORKERS_QUERY2 + WORKERS_QUERY3 + WORKERS_QUERY4))
      - IN_DEMUXERS=$WORKERS_QUERY1
      - IN_NAME="input"
    depends_on:
      rabbitmq:
        condition: service_healthy
    volumes:
      - ./cmd/inputBoundary/config.yaml:/config.yaml"

echo "
  outputBoundary:
    container_name: output
    image: output_boundary:latest
    entrypoint: /outputBoundary
    networks:
      - testing_net
    environment:
      - OUT_WORKERS=$((WORKERS_QUERY1 + WORKERS_QUERY2 + WORKERS_QUERY3 + WORKERS_QUERY4))
      - OUT_NAME="output"
    depends_on:
      rabbitmq:
        condition: service_healthy
    volumes:
      - ./cmd/outputBoundary/config.yaml:/config.yaml"

for ((n = 1; n <= WORKERS_HEARTBEATER; n++))
do
  echo "
  peer$n:
    container_name: peer$n
    image: invitation:latest
    entrypoint: /invitation
    networks:
      - testing_net
    environment:
      - INV_NAME=$n
      - INV_PEERS_COUNT=$WORKERS_HEARTBEATER
      - INV_CONTAINERS_DEMUX_COUNT=$WORKERS_QUERY1
      - INV_CONTAINERS_DISTANCE_COUNT=$WORKERS_QUERY2
      - INV_CONTAINERS_FASTEST_COUNT=$WORKERS_QUERY3
      - INV_CONTAINERS_AVERAGE_COUNT=$WORKERS_QUERY4
    volumes:
      - ./cmd/heartbeater/config.yaml:/config.yaml
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= WORKERS_QUERY1; n++))
do
    echo "
  demuxFilter$n:
    container_name: demux_filter$n
    image: demux_filter:latest
    entrypoint: /demuxFilter
    networks:
      - testing_net
    environment:
      - DEMUX_ID=$n
      - DEMUX_WORKERS_Q2=$WORKERS_QUERY2
      - DEMUX_WORKERS_Q3=$WORKERS_QUERY3
      - DEMUX_WORKERS_Q4=$WORKERS_QUERY4
      - DEMUX_NAME="demux_filter$n"
    volumes:
      - ./cmd/demuxFilter/config.yaml:/config.yaml
      - demuxState:/clients/
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= WORKERS_QUERY2; n++))
do
    echo "
  distanceFilter$n:
    container_name: distance_filter$n
    image: distance_filter:latest
    entrypoint: /distanceFilter
    networks:
      - testing_net
    environment:
      - DISTANCE_ID=$n
      - DISTANCE_DEMUXERS=$WORKERS_QUERY1
      - DISTANCE_NAME="distance_filter$n"
    volumes:
      - ./cmd/distanceFilter/config.yaml:/config.yaml
      - distanceState:/clients/
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= WORKERS_QUERY3; n++))
do
    echo "
  fastestFilter$n:
    container_name: fastest_filter$n
    image: fastest_filter:latest
    entrypoint: /fastestFilter
    networks:
      - testing_net
    environment:
      - FAST_ID=$n
      - FAST_DEMUXERS=$WORKERS_QUERY1
      - FAST_NAME="fastest_filter$n"
    volumes:
      - ./cmd/fastestFilter/config.yaml:/config.yaml
      - fastestState:/clients/
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= WORKERS_QUERY4; n++))
do
    echo "
  avgFilter$n:
    container_name: avg_filter$n
    image: avg_filter:latest
    entrypoint: /avgFilter
    networks:
      - testing_net
    environment:
      - AVG_ID=$n
      - AVG_DEMUXERS=$WORKERS_QUERY1
      - AVG_NAME="avg_filter$n"
    volumes:
      - ./cmd/avgFilter/config.yaml:/config.yaml
      - avgState:/clients/
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

echo '
volumes:
  distanceState:
  fastestState:
  avgState:
  demuxState:

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24'
