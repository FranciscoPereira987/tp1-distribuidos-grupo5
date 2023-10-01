SHELL := /bin/bash
PWD := $(shell pwd)

GIT_REMOTE = github.com/franciscopereira987/tp1-distribuidos.git

CMD ?= hello


default: build


build: GOOS=linux go build -0 ./bin/ ./cmd/$(CMD)/
.PHONY: build

build-image: 
	docker build -t $(CMD) -f cmd/$(CMD)/Dockerfile .
.PHONY: build-image

fmt:
	go fmt $(PWD)/...
.PHONY: fmt

test:
	go test $(PWD)/... -timeout 10s
.PHONY: test

docker-compose-up: build-image
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