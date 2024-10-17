FROM gcr.io/distroless/static:latest@sha256:ce46866b3a5170db3b49364900fb3168dc0833dfb46c26da5c77f22abb01d8c3
WORKDIR /
COPY manager manager
USER 65532:65532

# User env is required by opentelemetry-go
ENV USER=growthbook-controller

ENTRYPOINT ["/manager"]
