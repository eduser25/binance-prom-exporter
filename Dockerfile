FROM golang:latest

WORKDIR /go/src/binance-prom-exporter
COPY . .
RUN go build -ldflags "-linkmode external -extldflags -static" -a .

FROM gcr.io/distroless/static
COPY --from=0 /go/src/binance-prom-exporter/binance-prom-exporter /binance-prom-exporter
ENTRYPOINT ["/binance-prom-exporter"]
