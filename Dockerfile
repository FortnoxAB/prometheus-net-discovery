FROM golang:latest as builder

WORKDIR /build

COPY . .
RUN go get -v -t -d ./...
RUN CGO_ENABLED=0 go build -o prometheus-net-discovery .


FROM alpine:latest
WORKDIR /opt/net-discovery

COPY --from=builder /build/prometheus-net-discovery .

EXPOSE 9103
# change the subnet name
CMD ["./prometheus-net-discovery", "-networks", "x.x.x.x/x", "--filesdpath", "/opt/net-discovery/"]

