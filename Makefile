.PHONY:	mocks
SHELL := /bin/bash
MAKEFLAGS += --no-print-directory

IMAGE = quay.fnox.se/fortnox/prometheus-net-discovery
VERSION?=0.0.1-local

-include .env
export

build:
	CGO_ENABLED=0 GOOS=linux go build

docker: build
	docker build --pull --rm -t $(IMAGE):$(VERSION) .

push: docker
	docker push $(IMAGE):$(VERSION)

localrun:
	go run . 

test:
	go test ./... -count=1 -cover
