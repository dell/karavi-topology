.PHONY: all
all: help

help:
	@echo
	@echo "The following targets are commonly used:"
	@echo
	@echo "build    - Builds the code locally"
	@echo "clean    - Cleans the local build"
	@echo "podman   - Builds Podman image"
	@echo "tag      - Tags Podman image"
	@echo "push     - Pushes Podman image to a registry"
	@echo "check    - Runs code checking tools: lint, format, gosec, and vet"
	@echo "test     - Runs the unit tests"
	@echo

.PHONY: build
build: generate
	CGO_ENABLED=0 GOOS=linux go build -o ./cmd/topology/bin/service ./cmd/topology

.PHONY: clean
clean:
	rm -rf cmd/bin

.PHONY: generate
generate:
	go generate ./...

.PHONY: test
test:
	go test -count=1 -cover -race -timeout 30s -short ./...

.PHONY: build-base-image
build-base-image: download-csm-common
	$(eval include csm-common.mk)
	@echo "Building base image from $(DEFAULT_BASEIMAGE) and loading dependencies..."
	./scripts/build_ubi_micro.sh $(DEFAULT_BASEIMAGE)
	@echo "Base image build: SUCCESS"
	$(eval BASEIMAGE=topology-ubimicro:latest)

# Pre-requisites: RHEL, buildah, podman
.PHONY: podman
podman: build-base-image
	podman build $(NOCACHE) -t csm-topology -f Dockerfile --build-arg BASEIMAGE=$(BASEIMAGE) --build-arg GOIMAGE=$(DEFAULT_GOIMAGE) .

.PHONY: podman-no-cache
podman-no-cache:
	@make podman NOCACHE="--no-cache"

.PHONY: tag
tag:
	podman tag csm-topology:latest ${DOCKER_REPO}/csm-topology:latest

.PHONY: push
push:
	podman push ${DOCKER_REPO}/csm-topology:latest

.PHONY: check
check:
	./scripts/check.sh ./cmd/... ./internal/...

.PHONY: download-csm-common
download-csm-common:
	curl -O -L https://raw.githubusercontent.com/dell/csm/main/config/csm-common.mk
