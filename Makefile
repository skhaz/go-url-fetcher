.PHONY: compose fmt vet test

.SILENT:

SHELL:=/bin/bash

FILES=$(shell go list ./... | grep -v /vendor/)

compose: vet
	docker-compose up --build

fmt:
	go fmt
	go mod tidy

vet: fmt
	go vet

test: vet
	go test