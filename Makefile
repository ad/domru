.EXPORT_ALL_VARIABLES:
.DEFAULT_GOAL := default
DOCKER_SCAN_SUGGEST = false
CWD = $(shell pwd)
SRC_DIRS := .
GOCACHE ?= $$(pwd)/.go/cache
BUILD_VERSION=$(shell cat config.json | awk 'BEGIN { FS="\""; RS="," }; { if ($$2 == "version") {print $$4} }')
IMAGE ?= danielapatin/domru:${BUILD_VERSION}

export DOCKER_CLI_EXPERIMENTAL=enabled

.PHONY: build # Build the container image
build:
	@docker buildx create --use --name=crossplat --node=crossplat && \
	docker buildx build \
	    --build-arg="BUILD_VERSION=${BUILD_VERSION}" \
		--output "type=docker,push=false" \
		--tag $(IMAGE) \
		--tag danielapatin/domru:latest \
		.

.PHONY: publish # Push the image to the remote registry
publish:
	@docker buildx create --use --name=crossplat --node=crossplat && \
	docker buildx build \
		--platform linux/386,linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64 \
		--build-arg="BUILD_VERSION=${BUILD_VERSION}" \
		--output "type=image,push=true" \
		--tag $(IMAGE) \
		--tag danielapatin/domru:latest \
		.

lint:
	@docker run --rm -t -w $(CWD) -v $(CURDIR):$(CWD) -e GOFLAGS=-mod=vendor golangci/golangci-lint:latest golangci-lint run -v
