FROM golang:1.23.4-alpine AS builder

WORKDIR /app

COPY main.go go.mod go.sum ./
COPY internal/ ./internal/

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o checkmate

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/checkmate .
COPY examples/config.yaml /app/config.yaml

USER nonroot:nonroot

# Expose Prometheus/healthcheck port
EXPOSE 9100

ENTRYPOINT ["/app/checkmate", "/app/config.yaml"] 