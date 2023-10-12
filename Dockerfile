FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:21daf4c8bea788f6114822ab2d4a23cca6c682bdccc8aa7cae1124bcd8002066
LABEL vendor="Dell Inc." \
      name="csm-topology" \
      summary="Dell Container Storage Modules (CSM) for Observability - Topology" \
      description="Provides Kubernetes administrators with the topology data related to containerized storage that is provisioned by CSI (Container Storage Interface) Drivers for Dell storage products" \
      version="2.0.0" \
      license="Apache-2.0"
ARG SERVICE
COPY $SERVICE/bin/service /service
ENTRYPOINT ["/service"]
