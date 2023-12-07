#!/usr/bin/env bash

HB=${HB:-3}
Q1=${Q1:-1}
Q2=${Q2:-1}
Q3=${Q3:-1}
Q4=${Q4:-1}

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
      - IN_WORKERS=$((Q1 + Q2 + Q3 + Q4))
      - IN_DEMUXERS=$Q1
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
      - OUT_WORKERS=$((Q1 + Q2 + Q3 + Q4))
      - OUT_NAME="output"
    depends_on:
      rabbitmq:
        condition: service_healthy
    volumes:
      - ./cmd/outputBoundary/config.yaml:/config.yaml"

for ((n = 1; n <= HB; n++))
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
      - INV_PEERS_COUNT=$HB
      - INV_CONTAINERS_DEMUX_COUNT=$Q1
      - INV_CONTAINERS_DISTANCE_COUNT=$Q2
      - INV_CONTAINERS_FASTEST_COUNT=$Q3
      - INV_CONTAINERS_AVERAGE_COUNT=$Q4
    volumes:
      - ./cmd/heartbeater/config.yaml:/config.yaml
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= Q1; n++))
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
      - DEMUX_WORKERS_Q2=$Q2
      - DEMUX_WORKERS_Q3=$Q3
      - DEMUX_WORKERS_Q4=$Q4
      - DEMUX_NAME="demux_filter$n"
    volumes:
      - ./cmd/demuxFilter/config.yaml:/config.yaml
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= Q2; n++))
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
      - DISTANCE_DEMUXERS=$Q1
      - DISTANCE_NAME="distance_filter$n"
    volumes:
      - ./cmd/distanceFilter/config.yaml:/config.yaml
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= Q3; n++))
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
      - FAST_DEMUXERS=$Q1
      - FAST_NAME="fastest_filter$n"
    volumes:
      - ./cmd/fastestFilter/config.yaml:/config.yaml
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

for ((n = 1; n <= Q4; n++))
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
      - AVG_DEMUXERS=$Q1
      - AVG_NAME="avg_filter$n"
    volumes:
      - ./cmd/avgFilter/config.yaml:/config.yaml
    depends_on:
      rabbitmq:
        condition: service_healthy"
done

echo '
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24'

mkdir -p bin
sed "
s/%I/${INTERVAL:-10s}/
s/%V/${VICTIMS:-$(((HB+Q1+Q2+Q3+Q4)/2))}/
" setup/maniac-template.bash >bin/maniac.bash
chmod +x bin/maniac.bash
