ARG BASEIMAGE

# Build the sdk binary
FROM golang:1.21 as builder

# Set envirment variable
ENV APP_NAME csm-topology
ENV CMD_PATH cmd/topology/main.go

# Copy application data into image
COPY . /go/src/$APP_NAME
WORKDIR /go/src/$APP_NAME

# Build the binary
RUN go install github.com/golang/mock/mockgen@v1.6.0
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o /go/src/service /go/src/$APP_NAME/$CMD_PATH

# Build the sdk image
FROM $BASEIMAGE as final
LABEL vendor="Dell Inc." \
      name="csm-topology" \
      summary="Dell Container Storage Modules (CSM) for Observability - Metrics for Topology" \
      description="Provides Kubernetes administrators with the topology data related to containerized storage that is provisioned by CSI (Container Storage Interface) Drivers for Dell storage products" \
      version="2.0.0" \
      license="Apache-2.0"

COPY --from=builder /go/src/service /
ENTRYPOINT ["/service"]