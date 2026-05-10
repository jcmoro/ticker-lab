FROM golang:1.25-alpine AS builder
# go.work is intentionally NOT copied: it references modules outside the
# Docker build context. The `replace` directive in each go.mod resolves
# the shared httpx package locally inside the image.
WORKDIR /app
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/esios-go/go.mod apps/esios-go/go.sum* ./apps/esios-go/
WORKDIR /app/apps/esios-go
RUN go mod download
COPY apps/esios-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/esios-go .

FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/esios-go/go.mod apps/esios-go/go.sum* ./apps/esios-go/
WORKDIR /app/apps/esios-go
RUN go mod download
COPY apps/esios-go/*.go ./
COPY apps/esios-go/testdata/ ./testdata/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o ./esios-go .
EXPOSE 8120
CMD ["./esios-go"]

FROM alpine:3.21 AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/esios-go .
EXPOSE 8120
USER nobody
CMD ["./esios-go"]
