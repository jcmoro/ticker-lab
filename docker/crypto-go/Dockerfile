FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/crypto-go/go.mod apps/crypto-go/go.sum ./apps/crypto-go/
WORKDIR /app/apps/crypto-go
RUN go mod download
COPY apps/crypto-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/crypto-go .

FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/crypto-go/go.mod apps/crypto-go/go.sum ./apps/crypto-go/
WORKDIR /app/apps/crypto-go
RUN go mod download
COPY apps/crypto-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o ./crypto-go .
EXPOSE 8090
CMD ["./crypto-go"]

FROM alpine:3.21 AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/crypto-go .
EXPOSE 8090
USER nobody
CMD ["./crypto-go"]
