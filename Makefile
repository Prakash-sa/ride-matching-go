.PHONY: build run test docker

build:
	go build -o bin/ride-matching ./cmd/server

run:
	HTTP_ADDR=:8080 go run ./cmd/server

test:
	go test ./... -v

docker:
	docker build -t ride-matching:local .

compose-up:
	docker compose up -d redis postgres redpanda prometheus grafana

run-consumer:
	KAFKA_BROKERS=localhost:9092 REDIS_ADDR=localhost:6379 go run ./cmd/consumer
