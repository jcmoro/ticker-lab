FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work ./
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/cnmv-go/go.mod apps/cnmv-go/go.sum* ./apps/cnmv-go/
WORKDIR /app/apps/cnmv-go
RUN go mod download
COPY apps/cnmv-go/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/cnmv-go .

FROM golang:1.25-alpine AS dev
WORKDIR /app
COPY go.work ./
COPY apps/internal/httpx/ ./apps/internal/httpx/
COPY apps/cnmv-go/go.mod apps/cnmv-go/go.sum* ./apps/cnmv-go/
WORKDIR /app/apps/cnmv-go
RUN go mod download
COPY apps/cnmv-go/*.go ./
COPY apps/cnmv-go/testdata/ ./testdata/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/cnmv-go .
EXPOSE 8130
CMD ["./cnmv-go"]

FROM alpine:3.21 AS prod
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/cnmv-go .
EXPOSE 8130
USER nobody
CMD ["./cnmv-go"]
