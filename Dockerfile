FROM golang:1.23.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o checkmate

# Final stage using distroless
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/checkmate .

USER nonroot:nonroot

# Expose Prometheus/healthcheck port
EXPOSE 9100

ENTRYPOINT ["/app/checkmate"] 