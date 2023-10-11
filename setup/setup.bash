#!/usr/bin/env bash

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
  server:
    container_name: server
    image: server:latest
    entrypoint: /server
    networks:
      - testing_net
    environment:
      - IFZ_WORKERS_FIRST=1
      - IFZ_WORKERS_SECOND=${WORKERS_QUERY2}
      - IFZ_WORKERS_THIRD=${WORKERS_QUERY3}
      - IFZ_WORKERS_FOURTH=${WORKERS_QUERY4}
    depends_on:
      rabbitmq:
          condition: service_healthy
    volumes:
      - ./cmd/interface/config:/config"

for ((n = 1; n <= WORKERS_QUERY2; n++))
do
    echo "
  distanceFilter$n:
    container_name: distance_filter$n
    image: distance_filter:latest
    entrypoint: /filter
    networks:
      - testing_net
    environment:
      - DISTANCE_ID=$n
    volumes:
      - ./cmd/filtroDistancia/config:/config
    depends_on:
      - server"
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
    volumes:
      - ./cmd/fastestFilter:/cmd/fastestFilter
    depends_on:
      - server"
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
    volumes:
      - ./cmd/avgFilter/config.yaml:/config.yaml
    depends_on:
      - server"
done

echo '
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24'
