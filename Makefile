.PHONY:	mocks
SHELL := /bin/bash
MAKEFLAGS += --no-print-directory

IMAGE = quay.fnox.se/fortnox/prometheus-net-discovery
VERSION?=0.0.1-local

# Sets DOCKER_HOST to the Podman socket if it exists, otherwise defaults to the Docker socket
export DOCKER_HOST := unix://$(or $(wildcard /run/user/$(shell id -u)/podman/podman.sock),/var/run/docker.sock)


build:
	CGO_ENABLED=0 GOOS=linux go build

docker: build
	docker build --pull --rm -t $(IMAGE):$(VERSION) .

push: docker
	docker push $(IMAGE):$(VERSION)

localrun:
	go run main.go

test:
	go test ./... -count=1 -cover

test-coverage:
	go test ./... -count=1 -coverprofile=coverage.out
	grep -vE "/mocks/|main.go" coverage.out > coverage-filtered.out
	go tool cover -func=coverage-filtered.out

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi
