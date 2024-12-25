# Go parameters
BINARY_NAME=checkmate
MAIN_PATH=./main.go
DOCKER_IMAGE=checkmate
VERSION?=1.0.0

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

.PHONY: all build clean test coverage lint deps docker-build docker-run docker-push help

all: clean lint test build

build:
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

test:
	$(GOTEST) -v ./...

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

lint:
	$(GOLINT) run

deps:
	$(GOMOD) download
	$(GOMOD) tidy

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run:
	docker run -p 9100:9100 $(DOCKER_IMAGE):$(VERSION)

docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

dev: deps lint test build

help:
	@echo "Available targets:"
	@echo "  all          : Clean, lint, test, and build"
	@echo "  build        : Build the application"
	@echo "  clean        : Clean build files"
	@echo "  test         : Run tests"
	@echo "  coverage     : Run tests with coverage report"
	@echo "  lint         : Run linter"
	@echo "  deps         : Download dependencies"
	@echo "  dev          : Setup development environment"
	@echo "  docker-build : Build Docker image"
	@echo "  docker-run   : Run Docker container"
	@echo "  docker-push  : Push Docker image"
	@echo "  help         : Show this help message" 