FROM gcr.io/distroless/static:latest@sha256:69830f29ed7545c762777507426a412f97dad3d8d32bae3e74ad3fb6160917ea
WORKDIR /
COPY manager manager
USER 65532:65532

# User env is required by opentelemetry-go
ENV USER=growthbook-controller

ENTRYPOINT ["/manager"]
