FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY apps/crypto-go/go.mod apps/crypto-go/go.sum ./
RUN go mod download
COPY apps/crypto-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o crypto-go .

FROM alpine:3.21 AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/crypto-go .
EXPOSE 8090
USER nobody
CMD ["./crypto-go"]
