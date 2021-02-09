FROM golang:latest

WORKDIR /go/src/binance-pricer
COPY . .
RUN go build -ldflags "-linkmode external -extldflags -static" -a .

FROM gcr.io/distroless/static
COPY --from=0 /go/src/binance-pricer/binance-pricer /binance-pricer
ENTRYPOINT ["/binance-pricer"]
