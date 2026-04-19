FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY apps/converter-go/go.mod apps/converter-go/go.sum ./
RUN go mod download
COPY apps/converter-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o converter-go .

FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY apps/converter-go/go.mod apps/converter-go/go.sum ./
RUN go mod download
COPY apps/converter-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o converter-go .
EXPOSE 8080
CMD ["./converter-go"]

FROM alpine:3.21 AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/converter-go .
EXPOSE 8080
USER nobody
CMD ["./converter-go"]
