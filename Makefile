SHELL := /bin/bash
PWD := $(shell pwd)

GIT_REMOTE = github.com/franciscopereira987/tp1-distribuidos

CMD := cliente interface filtroDistancia fastestFilter avgFilter
BIN := $(addprefix bin/,$(CMD))

all: $(BIN)
.PHONY: all

$(BIN):
	go build -o $@ $(GIT_REMOTE)/cmd/$(notdir $@)

build-image:
	docker build -t server -f cmd/interface/Dockerfile .
	docker build -t distance_filter -f cmd/filtroDistancia/Dockerfile .
	docker build -t fastest_filter -f cmd/fastestFilter/Dockerfile .
	docker build -t avg_filter -f cmd/avgFilter/Dockerfile .
	docker build -t client -f cmd/cliente/Dockerfile .
.PHONY: build-image

fmt:
	go fmt $(PWD)/...
.PHONY: fmt

test:
	go test $(PWD)/... -timeout 10s
.PHONY: test

setup:
	./setup/setup.bash > docker-compose-dev.yaml
.PHONY: setup

run-client:
	docker run --rm -v ./client:/client --network tp1_testing_net --entrypoint /cliente client
.PHONY: run-client

docker-compose-up:
	# Hay que agregar la creacion de todas las imagenes
	# Supongo que estan creadas
	docker compose -f docker-compose-dev.yaml up -d
.PHONY: docker-compose-up

docker-compose-down:
	docker compose -f docker-compose-dev.yaml stop
	docker compose -f docker-compose-dev.yaml down
.PHONY: docker-compose-down

docker-compose-logs:
	docker compose -f docker-compose-dev.yaml logs -f
.PHONY: docker-compose-logs
