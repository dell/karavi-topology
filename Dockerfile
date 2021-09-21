FROM scratch
LABEL vendor="Dell Inc." \
      name="csm-topology" \
      summary="Dell EMC Container Storage Modules (CSM) for Observability - Topology" \
      description="Provides Kubernetes administrators with the topology data related to containerized storage that is provisioned by CSI (Container Storage Interface) Drivers for Dell EMC storage products" \
      version="2.0.0" \
      license="Apache-2.0"
ARG SERVICE
COPY $SERVICE/bin/service /service
ENTRYPOINT ["/service"]
