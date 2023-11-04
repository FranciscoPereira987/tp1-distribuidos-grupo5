#!/usr/bin/env bash

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
    depends_on:
      rabbitmq:
        condition: service_healthy
    volumes:
      - ./cmd/outputBoundary/config.yaml:/config.yaml"

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
      - DEMUX_WORKERS_Q2=$WORKERS_QUERY2
      - DEMUX_WORKERS_Q3=$WORKERS_QUERY3
      - DEMUX_WORKERS_Q4=$WORKERS_QUERY4
    volumes:
      - ./cmd/demuxFilter/config.yaml:/config.yaml
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
      - DISTANCE_DEMUXERS=$WORKERS_QUERY1
    volumes:
      - ./cmd/distanceFilter/config.yaml:/config.yaml
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
    volumes:
      - ./cmd/fastestFilter/config.yaml:/config.yaml
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
