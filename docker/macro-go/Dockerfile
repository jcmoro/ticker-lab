FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/macro-go/go.mod apps/macro-go/go.sum ./apps/macro-go/
WORKDIR /app/apps/macro-go
RUN go mod download
COPY apps/macro-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/macro-go .

FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/macro-go/go.mod apps/macro-go/go.sum ./apps/macro-go/
WORKDIR /app/apps/macro-go
RUN go mod download
COPY apps/macro-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o ./macro-go .
EXPOSE 8110
CMD ["./macro-go"]

FROM alpine:3.21 AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/macro-go .
EXPOSE 8110
USER nobody
CMD ["./macro-go"]
