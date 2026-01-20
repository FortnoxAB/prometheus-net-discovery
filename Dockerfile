FROM gcr.io/distroless/static-debian13:nonroot
COPY prometheus-net-discovery /
USER nonroot
ENTRYPOINT ["/prometheus-net-discovery"]
