SHELL := /bin/bash

GIT_REMOTE := github.com/franciscopereira987/tp1-distribuidos.git
CLIENT_ID ?= c1

CMD := client inputBoundary outputBoundary demuxFilter distanceFilter fastestFilter avgFilter heartbeater
BIN := $(addprefix bin/,$(CMD))

all: $(BIN)
.PHONY: all

$(BIN):
	go build -o $@ ./cmd/$(notdir $@)
.PHONY: $(BIN)

clean:
	rm -f $(BIN)
.PHONY: clean

build-image:
	docker build -t input_boundary -f cmd/inputBoundary/Dockerfile .
	docker build -t output_boundary -f cmd/outputBoundary/Dockerfile .
	docker build -t demux_filter -f cmd/demuxFilter/Dockerfile .
	docker build -t distance_filter -f cmd/distanceFilter/Dockerfile .
	docker build -t fastest_filter -f cmd/fastestFilter/Dockerfile .
	docker build -t avg_filter -f cmd/avgFilter/Dockerfile .
	docker build -t client -f cmd/client/Dockerfile .
	docker build -t invitation -f cmd/heartbeater/Dockerfile .
.PHONY: build-image

fmt:
	go fmt ./...
.PHONY: fmt

test:
	go test ./... -timeout 10s
.PHONY: test

setup: docker-compose-dev.yaml
	setup/setup.bash > $^
.PHONY: setup

run-client:
	docker run --rm -v ./clients/$(CLIENT_ID):/client  --network tp1_testing_net --entrypoint /cliente client
	@printf '%s\n' ====================================== tests:
	@test/test.bash clients/$(CLIENT_ID)
.PHONY: run-client

run-maniac:
	bin/maniac.bash
.PHONY: run-maniac

docker-compose-up:
	docker compose -f docker-compose-dev.yaml up -d
.PHONY: docker-compose-up

docker-compose-down:
	docker compose -f docker-compose-dev.yaml stop
	docker compose -f docker-compose-dev.yaml down
.PHONY: docker-compose-down

docker-compose-logs:
	docker compose -f docker-compose-dev.yaml logs -f
.PHONY: docker-compose-logs
