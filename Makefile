.PHONY: all
all: help

help:
	@echo
	@echo "The following targets are commonly used:"
	@echo
	@echo "build    - Builds the code locally"
	@echo "clean    - Cleans the local build"
	@echo "docker   - Builds Docker image"
	@echo "tag      - Tags Docker image"
	@echo "push     - Pushes Docker image to a registry"
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

.PHONY: docker
docker:
	SERVICE=cmd/topology docker build -t karavi-topology -f Dockerfile cmd/topology

.PHONY: tag
tag:
	docker tag karavi-topology:latest ${DOCKER_REPO}/karavi-topology:latest

.PHONY: push
push:
	docker push ${DOCKER_REPO}/karavi-topology:latest

.PHONY: check
check:
	./scripts/check.sh ./cmd/... ./internal/...

